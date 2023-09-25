// the data of the file->this service->1. divide the file into chunks (communicate with chunk_storage_service) 2. record the hash of those chunks (on chaincode)

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"encoding/json"
	"os"
	"flag"
	"path/filepath"
	"io/ioutil"
	"math/big"

	utils "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/utils"
	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

var (
	port = flag.String("port", ":50051", "listening port")
	contract *gateway.Contract
)

type Slot struct {
	StartSlot    int     `json:"startSlot"`
	EndSlot      int     `json:"endSlot"`
}

type HashSlotTable struct {
	HST map[string]Slot `json:"hashSlotTable"`
}

type server struct{
	pb.UnimplementedFilePartitionServer
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func initializeSmartContract() {
	fmt.Println("Initialize smart contract...")
	err := os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	if err != nil {
		log.Fatalf("Error setting DISCOVERY_AS_LOCALHOST environment variable: %v", err)
	}

	walletPath := "wallet"
	// remove any existing wallet from prior runs
	os.RemoveAll(walletPath)
	wallet, err := gateway.NewFileSystemWallet(walletPath)
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			log.Fatalf("Failed to populate wallet contents: %v", err)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"connection-org1.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gw.Close()

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}

	chaincodeName := "basic"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	contract = network.GetContract(chaincodeName)
}

func getOrgID(chunkHash string, hashSlotTable HashSlotTable) string {
	var orgID string
	hashInt := new(big.Int)
	hashInt.SetString(chunkHash, 16)
	hashMod := hashInt.Mod(hashInt, big.NewInt(int64(utils.NumOfSlots)))
	slotID := int(hashMod.Int64())
	for org, slot := range hashSlotTable.HST {
		if slotID >= slot.StartSlot && slotID <= slot.EndSlot {
			orgID = org
		}
	}
	return orgID
}

func storeChunk(request *pb.FilePartitionRequest) {
	fileObj := File{}

	// Accept file from remote nodes
	fileContent := make([]byte, len(request.Data))
	copy(fileContent, request.Data)
	fileHash := utils.GetHash(fileContent)
	fileObj.SetHashValue(fileHash)

	res, err := contract.EvaluateTransaction("GetHashSlotTable")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v", err)
	}

	var hashSlotTable HashSlotTable

	err = json.Unmarshal(res, &hashSlotTable)
	if err != nil {
		return
	}

	// Divide file into stripes
	for i := 0; i < len(fileContent); i += utils.StripeSize {
		data := fileContent[i:min(i+utils.StripeSize, len(fileContent))]
		if len(data) < utils.StripeSize {
			data = append(data, make([]byte, utils.StripeSize-len(data))...)
		}

		encodedChunks, err := utils.Encode(utils.N, utils.K, data)
		if err != nil {
			log.Fatalf("Failed to encode file: %v", err)
		}

		stripeObj := Stripe{}
		stripeObj.SetHashValue(utils.GetHash(data))
		nodeCounts := make(map[string]int)
		for _, node := range utils.MasterNodes {
			nodeCounts[node] = 0;
		}
		theoricalMap := make(map[string]string)
		actualMap := make(map[string]string)
		desiredCount := utils.N/utils.L

		for _, chunk := range encodedChunks {
			chunkHash := utils.GetHash(chunk)
			id := getOrgID(chunkHash, hashSlotTable)
			theoricalMap[chunkHash] = id
			actualMap[chunkHash] = id
			nodeCounts[id]++
		}

		for _, node := range utils.MasterNodes {
			for hash, currentNode := range actualMap {
				if currentNode == node && nodeCounts[node] > desiredCount {
					// Find the next node in the sequence to transfer the hash entry
					nextNode := ""
					for _, n := range utils.MasterNodes {
						if n != node && nodeCounts[n] < desiredCount {
							nextNode = n
							break
						}
					}
					if nextNode != "" {
						// Move the hash entry to the next node in the sequence
						actualMap[hash] = nextNode
						nodeCounts[node]--
						nodeCounts[nextNode]++
					}
				}
			}
		}

		// fmt.Println("****** Theorical ******")
		// fmt.Println(theoricalMap)
		// fmt.Println("******   Actual  ******")
		// fmt.Println(actualMap)

		for _, chunk := range encodedChunks {
			chunkHash := utils.GetHash(chunk)
			chunkObj := Chunk{}
			chunkObj.SetHashValue(chunkHash)
			stripeObj.AddChunk(chunkObj)

			// Store chunk in a file
			conn, err := grpc.Dial(actualMap[chunkHash], grpc.WithInsecure())
			if err != nil {
				log.Fatalf("Failed to connect: %v", err)
			}
			defer conn.Close()

			// create a stub
			stub := pb.NewChunkStorageClient(conn)

			// create a request object
			rq := &pb.ChunkStorageRequest{
				Data:  chunk,
			}

			// send the request and get the response
			_, err = stub.StoreChunk(context.Background(), rq)
			if err != nil {
				log.Fatalf("Failed to store chunk: %v", err)
			}

			if actualMap[chunkHash] != theoricalMap[chunkHash] {
				nConn, err := grpc.Dial(theoricalMap[chunkHash], grpc.WithInsecure())
				if err != nil {
					log.Fatalf("Failed to connect: %v", err)
				}
				defer nConn.Close()

				nStub := pb.NewChunkStorageClient(nConn)

				nRq := &pb.LinkStorageRequest{
					Hash:  chunkHash,
					Id: actualMap[chunkHash],
				}

				_, err = nStub.StoreLink(context.Background(), nRq)
				if err != nil {
					log.Fatalf("Failed to store link: %v", err)
				}
			}
		}

		fileObj.AddStripe(stripeObj)
	}

	// Marshal fileObj to json and print it to a file
	jsonFile, err := json.MarshalIndent(fileObj, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal file object: %v", err)
	}
	fileName := fileObj.FileHash + ".json"
	err = ioutil.WriteFile(fileName, jsonFile, 0644)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	result, err := contract.SubmitTransaction("StoreFileTree", fileObj.FileHash, string(jsonFile))
	if err != nil {
		log.Fatalf("Failed to Submit transaction: %v", err)
	}

	fmt.Println(string(result))
	fmt.Println("--------- File partition end ---------")

	// Example of getting stored file tree
	// result, err = contract.EvaluateTransaction("GetHashSlotTable", "9")//fileObj.FileHash, "client 1")
	// if err != nil {
	// 	log.Fatalf("Failed to evaluate transaction: %v", err)
	// }
	// fmt.Printf("The file object: \n", string(result))

	// result, err = contract.EvaluateTransaction("GetFileTree", fileObj.FileHash)//fileObj.FileHash, "client 1")
	// if err != nil {
	// 	log.Fatalf("Failed to evaluate transaction: %v", err)
	// }
	// fmt.Printf("The file object: \n", string(result))
	// Example of unmarshalling b to a json object
	// b := []byte(result)
	// var nFileObj File
	// err = json.Unmarshal(b, &nFileObj)
	// if err != nil {
	// 	log.Fatalf("Failed to unmarshal json: %v", err)
	// }
	// fmt.Printf("The file hash: %s\n", nFileObj.FileHash)
}

func (s *server) PartitionFile(ctx context.Context, request *pb.FilePartitionRequest) (*pb.FilePartitionResponse, error) {
	fmt.Println("---------File partition start---------")
	fileObj := File{}

	// Accept file from remote nodes
	fileContent := make([]byte, len(request.Data))
	copy(fileContent, request.Data)
	fileHash := utils.GetHash(fileContent)
	fileObj.SetHashValue(fileHash)
	fmt.Println("File hash:", fileHash)

	// Divide file into stripes
	for i := 0; i < len(fileContent); i += utils.StripeSize {
		data := fileContent[i:min(i+utils.StripeSize, len(fileContent))]
		if len(data) < utils.StripeSize {
			data = append(data, make([]byte, utils.StripeSize-len(data))...)
		}

		encodedChunks, err := utils.Encode(utils.N, utils.K, data)
		if err != nil {
			log.Fatalf("Failed to encode file: %v", err)
		}

		stripeObj := Stripe{}
		stripeObj.SetHashValue(utils.GetHash(data))
		for _, chunk := range encodedChunks {
			chunkHash := utils.GetHash(chunk)
			chunkObj := Chunk{}
			chunkObj.SetHashValue(chunkHash)
			stripeObj.AddChunk(chunkObj)

			if _, err := os.Stat("memory"); os.IsNotExist(err) {
				os.Mkdir("memory", 0755)
			}
			
			// Store the chunk in local memory folder
			err := ioutil.WriteFile(fmt.Sprintf("memory/%s", chunkHash), chunk, 0644)
			if err != nil {
				log.Fatalf("Failed to store chunk: %v", err)
			}
		}

		fileObj.AddStripe(stripeObj)
	}

	go storeChunk(request)

	return &pb.FilePartitionResponse{Status: fileHash}, nil
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./file_partition_service [-h] [-port string]")
		flag.PrintDefaults()
	}
	flag.Parse()

	initializeSmartContract()

	// Create a listener on the TCP port
	lis, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024 * 1024 * 1024), // Set the maximum receive message size (in this case, 10MB)
	}

	// Create a new gRPC server
	s := grpc.NewServer(opts...)

	// Register the chunk storage server
	pb.RegisterFilePartitionServer(s, &server{})

	// Start the server
	fmt.Printf("Starting chunk storage server on port %s\n", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func populateWallet(wallet *gateway.Wallet) error {
	credPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"users",
		"User1@org1.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := os.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org1MSP", string(cert), string(key))

	return wallet.Put("appUser", identity)
}

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"encoding/json"
	"os"
	"path/filepath"

	utils "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/utils"
	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

const (
	port = ":50051"
)

var contract *gateway.Contract

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

func (s *server) PartitionFile(ctx context.Context, request *pb.FilePartitionRequest) (*pb.FilePartitionResponse, error) {
	fmt.Println("Start partitioning a file...")
	fileObj := File{}

	// Accept file from remote nodes
	fileContent := request.Data
	fileObj.SetHashValue(utils.GetHash(fileContent))

	// Divide file into stripes
	// var stripes []Stripe
	for i := 0; i < len(fileContent); i += utils.StripeSize {
		data := fileContent[i:min(i+utils.StripeSize, len(fileContent))]
		if len(data) < utils.StripeSize {
			data = append(data, make([]byte, utils.StripeSize-len(data))...)
		}
		// stripes = append(stripes, stripe)
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

			res, err := contract.EvaluateTransaction("QueryNodeID", chunkHash, utils.BlockHeight)
			if err != nil {
				log.Fatalf("Failed to evaluate transaction: %v", err)
			}
			nodeID := string(res)

			// Store chunk in a file
			conn, err := grpc.Dial(nodeID, grpc.WithInsecure())
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
			result, err := stub.StoreChunk(context.Background(), rq)
			if err != nil {
				log.Fatalf("Failed to store chunk: %v", err)
			}
			fmt.Printf("Store chunk %s to node %s %s\n", chunkHash, nodeID, result.Status)

		}

		fileObj.AddStripe(stripeObj)
	}

	// Marshal fileObj to json and print it to a file
	jsonFile, err := json.Marshal(fileObj)
	if err != nil {
		log.Fatalf("Failed to marshal file object: %v", err)
	}

	result, err := contract.SubmitTransaction("StoreFileTree", fileObj.FileHash, string(jsonFile))
	if err != nil {
		log.Fatalf("Failed to Submit transaction: %v", err)
	}
	// Example of getting stored file tree
	// result, err = contract.EvaluateTransaction("QueryFileTree", fileObj.FileHash)
	// if err != nil {
	// 	log.Fatalf("Failed to evaluate transaction: %v", err)
	// }
	// fmt.Printf("The file object: %s\n", result)
	// Example of unmarshalling b to a json object
	// b := []byte(result)
	// var nFileObj File
	// err = json.Unmarshal(b, &nFileObj)
	// if err != nil {
	// 	log.Fatalf("Failed to unmarshal json: %v", err)
	// }
	// fmt.Printf("The file hash: %s\n", nFileObj.FileHash)

	return &pb.FilePartitionResponse{Status: "SUCCESS"}, nil
}

func main() {
	initializeSmartContract()

	// Create a listener on the TCP port
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()

	// Register the chunk storage server
	pb.RegisterFilePartitionServer(s, &server{})

	// Start the server
	fmt.Printf("Starting chunk storage server on port %s\n", port)
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"flag"
	"path/filepath"

	utils "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/utils"
	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

var (
	fileHash = flag.String("hash", "", "the hash value of the requested file")
	pk = flag.String("pk", "client 1", "the public key of the client. Use `client 1` for simplicity")
	contract *gateway.Contract
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./request_file [-h] [-hash string] [-pk string]")
		flag.PrintDefaults()
	}
	flag.Parse()

	initializeSmartContract()
	// Example of getting stored file tree
	result, err := contract.EvaluateTransaction("QueryFileTree", *fileHash, *pk)
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v", err)
	}
	// Example of unmarshalling b to a json object
	b := []byte(result)
	var fileObj File
	err = json.Unmarshal(b, &fileObj)
	if err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}

	finalFile := []byte{}

	// Iterate through each stripe in the file object
	for _, stripe := range fileObj.StripeHashes {
		stripeBytes := make([][]byte, len(stripe.ChunkHashes))
		// Create a channel to synchronize the goroutines
		ch := make(chan []byte, len(stripe.ChunkHashes))
		defer close(ch)
		// Record the corresponding storage node and index for each chunk
		hashToID := make(map[string]string)
		hashToIndex := make(map[string][]int)
		for i, chunk := range stripe.ChunkHashes {
			chunkHash := chunk.ChunkHash
			nodeID, err := contract.EvaluateTransaction("QueryNodeID", chunkHash, utils.BlockHeight)
			if err != nil {
				log.Fatalf("Failed to evaluate transaction: %v", err)
			}

			hashToID[chunkHash] = string(nodeID)
			if _, ok := hashToIndex[chunkHash]; ok {
				hashToIndex[chunkHash] = append(hashToIndex[chunkHash], i)
			} else {
				hashToIndex[chunkHash] = []int{i}
			}
		}
		
		// Start N goroutines to get chunk data and stop when receiving K replies
		// var wg sync.WaitGroup
		// wg.Add(utils.N)
		for _, chunk := range stripe.ChunkHashes {
			chunkHash := chunk.ChunkHash
			go requestChunk(hashToID[chunkHash], chunkHash, ch)
		}

		for i := 0; i < utils.K; i++ {
			select {
			case data := <-ch:
				hash := utils.GetHash(data)
				for _, index := range hashToIndex[hash] {
					stripeBytes[index] = data
				}
			}
		}

		stripeData, err := utils.Decode(utils.N, utils.K, stripeBytes)
		if err != nil {
			log.Fatalf("Failed to decode: %v", err)
		}
		finalFile = append(finalFile, stripeData...)

		// wg.Wait()
		// defer close(ch)
	}


	// Write finalFile to a file named "out"
	err = ioutil.WriteFile("out", finalFile, 0644)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
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

func requestChunk(addr string, chunkHash string, ch chan []byte) {
	// defer wg.Done()
	// connect the remote node
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()
	stub := pb.NewChunkStorageClient(conn)
	// request chunk data from the remote node
	rq := &pb.ChunkRequest{
		Hash:  chunkHash,
	}
	chunkData, err := stub.GetChunk(context.Background(), rq)
	if err != nil {
		log.Fatalf("Failed to get chunk: %v", err)
	}
	ch <- chunkData.Data
}

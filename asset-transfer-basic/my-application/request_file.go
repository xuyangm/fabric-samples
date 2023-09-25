package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
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
	contract *gateway.Contract
)

type Slot struct {
	StartSlot    int     `json:"startSlot"`
	EndSlot      int     `json:"endSlot"`
}

type HashSlotTable struct {
	HST map[string]Slot `json:"hashSlotTable"`
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./request_file [-h] [-hash string]")
		flag.PrintDefaults()
	}
	flag.Parse()

	initializeSmartContract()
	result, err := contract.EvaluateTransaction("GetFileTree", *fileHash)
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v", err)
	}

	b := []byte(result)
	var fileObj File
	err = json.Unmarshal(b, &fileObj)
	if err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}

	result, err = contract.EvaluateTransaction("GetHashSlotTable")
	if err != nil {
		log.Fatalf("Failed to evaluate transaction: %v", err)
	}
	var tree HashSlotTable
	b = []byte(result)
	err = json.Unmarshal(b, &tree)
	if err != nil {
		log.Fatalf("Failed to unmarshal json: %v", err)
	}
	finalFile := []byte{}

	for _, stripe := range fileObj.StripeHashes {
		stripeBytes := make([][]byte, len(stripe.ChunkHashes))
		// Create a channel to synchronize the goroutines
		ch := make(chan []byte, len(stripe.ChunkHashes))
		defer close(ch)
		// Record the corresponding storage node and index for each chunk
		hashToID := make(map[string]string)
		hashToIndex := make(map[string][]int)

		// Calculate slot ID
		stripeHash := stripe.StripeHash
		hashInt := new(big.Int)
		hashInt.SetString(stripeHash, 16)
		hashMod := hashInt.Mod(hashInt, big.NewInt(int64(utils.NumOfSlots)))
		slotID := int(hashMod.Int64())
		var nodeID string
		for id, slot := range tree.HST {
			if slotID >= slot.StartSlot && slotID <= slot.EndSlot {
				nodeID = id
			}
		}

		for i, chunk := range stripe.ChunkHashes {
			chunkHash := chunk.ChunkHash

			hashToID[chunkHash] = nodeID
			if _, ok := hashToIndex[chunkHash]; ok {
				hashToIndex[chunkHash] = append(hashToIndex[chunkHash], i)
			} else {
				hashToIndex[chunkHash] = []int{i}
			}
		}
		
		// Start N goroutines to get chunk data and stop when receiving K replies
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

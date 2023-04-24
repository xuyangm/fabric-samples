package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"flag"

	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
)

var (
	address = flag.String("port", ":50051", "the port of remote server")
	fn = flag.String("fn", "mymodel.pth", "the name of the file to be stored")
	accessRule = flag.String("flag", "TFF", "enable access rules")
	permission = flag.String("allowed", "client 1", "allowed users")
	banned = flag.String("banned", "", "banned users")
	tokens = flag.Int("tokens", 0, "the number of tokens")
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./store_file [-h] [-port string] [-fn string] [-flag string] [-allowed string] [-banned string] [-tokens int]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Connect to the server
	conn, err := grpc.Dial(*address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a new client
	client := pb.NewFilePartitionClient(conn)

	// Read the chunk data from a file
	data, err := ioutil.ReadFile(*fn)
	if err != nil {
		log.Fatalf("Failed to read chunk data: %v", err)
	}

	// Calculate the hash of the chunk data
	hash := sha256.Sum256(data)
	hashString := hex.EncodeToString(hash[:])

	// Send the chunk to the server
	response, err := client.PartitionFile(context.Background(), &pb.FilePartitionRequest{
		Data: data,
		Flag: *accessRule,
		Permission: *permission,
		Banned: *banned,
		Token: int32(*tokens),
	})
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	// Check the response status
	if response.Status != "SUCCESS" {
		log.Fatalf("Failed to store chunk: %v", response.Status)
	}

	fmt.Printf("Stored a file with hash %s\n", hashString)
}
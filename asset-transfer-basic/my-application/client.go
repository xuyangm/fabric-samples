package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Connect to the server
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create a new client
	client := pb.NewFilePartitionClient(conn)

	// Read the chunk data from a file
	data, err := ioutil.ReadFile("abc.txt")
	if err != nil {
		log.Fatalf("Failed to read chunk data: %v", err)
	}

	// Calculate the hash of the chunk data
	hash := sha256.Sum256(data)
	hashString := hex.EncodeToString(hash[:])

	// Send the chunk to the server
	response, err := client.PartitionFile(context.Background(), &pb.FilePartitionRequest{
		Data: data,
	})
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	// Check the response status
	if response.Status != "SUCCESS" {
		log.Fatalf("Failed to store chunk: %v", response.Status)
	}

	fmt.Printf("Stored chunk with hash %s\n", hashString)
}
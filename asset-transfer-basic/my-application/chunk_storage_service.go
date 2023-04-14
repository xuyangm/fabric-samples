package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"

	"github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos/chunkstorage"
	"google.golang.org/grpc"
)

const (
	port = ":50052"
)

type server struct{}

func (s *server) StoreChunk(ctx context.Context, in *chunkstorage.ChunkStorageRequest) (*chunkstorage.ChunkStorageResponse, error) {
	// Store the chunk in memory folder
	hash := sha256.Sum256(in.GetData())
	hashString := hex.EncodeToString(hash[:])
	err := ioutil.WriteFile(fmt.Sprintf("memory/%s", hashString), in.GetData(), 0644)
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	// Return success response
	return &chunkstorage.ChunkStorageResponse{Status: "SUCCESS"}, nil
}

func main() {
	// Create a listener on the TCP port
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()

	// Register the chunk storage server
	chunkstorage.RegisterChunkStorageServer(s, &server{})

	// Start the server
	fmt.Printf("Starting chunk storage server on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}


package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"flag"

	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
)

var port = flag.String("port", ":50052", "listening port")

type server struct{
	pb.UnimplementedChunkStorageServer
}

func (s *server) StoreChunk(ctx context.Context, in *pb.ChunkStorageRequest) (*pb.ChunkStorageResponse, error) {
	// Ensure the memory directory exists
	if _, err := os.Stat("memory"); os.IsNotExist(err) {
		os.Mkdir("memory", 0755)
	}
	
	// Store the chunk in memory folder
	hash := sha256.Sum256(in.GetData())
	hashString := hex.EncodeToString(hash[:])
	err := ioutil.WriteFile(fmt.Sprintf("memory/%s", hashString), in.GetData(), 0644)
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	// Return success response
	return &pb.ChunkStorageResponse{Status: "SUCCESS"}, nil
}

func (s *server) GetChunk(ctx context.Context, in *pb.ChunkRequest) (*pb.ChunkResponse, error) {
	hashString := in.GetHash()
	// Search local folder "memory". If there is a file named as hashString, return it.
	data, err := ioutil.ReadFile(fmt.Sprintf("memory/%s", hashString))
	if err != nil {
		return nil, err
	}

	return &pb.ChunkResponse{Data: data}, nil
}


func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./chunk_storage_service [-h] [-port string]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Create a listener on the TCP port
	lis, err := net.Listen("tcp", *port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()

	// Register the chunk storage server
	pb.RegisterChunkStorageServer(s, &server{})

	// Start the server
	fmt.Printf("Starting chunk storage server on port %s\n", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}


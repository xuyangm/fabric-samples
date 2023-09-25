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
	"math/big"

	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
)

var port = flag.String("port", ":50052", "listening port")
var linkMap = make(map[string]string)

type server struct{
	pb.UnimplementedChunkStorageServer
}

func (s *server) StoreChunk(ctx context.Context, in *pb.ChunkStorageRequest) (*pb.ChunkStorageResponse, error) {
	// Ensure the memory directory exists
	if _, err := os.Stat("memory1"); os.IsNotExist(err) {
		os.Mkdir("memory1", 0755)
	}

	if _, err := os.Stat("memory2"); os.IsNotExist(err) {
		os.Mkdir("memory2", 0755)
	}
	
	// Store the chunk in memory folder
	hash := sha256.Sum256(in.GetData())
	hashString := hex.EncodeToString(hash[:])
	hashInt := new(big.Int)
	hashInt.SetString(hashString, 16)
	hashMod := hashInt.Mod(hashInt, big.NewInt(int64(16384)))
	slotID := int(hashMod.Int64())

	targetDirectory := "memory2"
	if slotID < 8193 {
		targetDirectory = "memory1"
	}

	err := ioutil.WriteFile(fmt.Sprintf("%s/%s", targetDirectory, hashString), in.GetData(), 0644)
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	// Return success response
	return &pb.ChunkStorageResponse{Status: "SUCCESS"}, nil
}

func (s *server) GetChunk(ctx context.Context, in *pb.ChunkRequest) (*pb.ChunkResponse, error) {
	hashString := in.GetHash()

	if _, ok := linkMap[hashString]; ok {
		nConn, err := grpc.Dial(linkMap[hashString], grpc.WithInsecure())
		if err != nil {
			log.Fatalf("Failed to connect: %v", err)
		}
		defer nConn.Close()

		nStub := pb.NewChunkStorageClient(nConn)

		nRq := &pb.ChunkRequest{
			Hash:  hashString,
		}
	
		chunkData, err := nStub.GetChunk(context.Background(), nRq)
		if err != nil {
			log.Fatalf("Failed to get chunk: %v", err)
		}
		return &pb.ChunkResponse{Data: chunkData.Data}, nil
	}

	
	hashInt := new(big.Int)
	hashInt.SetString(hashString, 16)
	hashMod := hashInt.Mod(hashInt, big.NewInt(int64(16384)))
	slotID := int(hashMod.Int64())

	targetDirectory := "memory2"
	if slotID < 8193 {
		targetDirectory = "memory1"
	}

	// Search local folder "memory". If there is a file named as hashString, return it.
	data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", targetDirectory, hashString))
	if err != nil {
		return nil, err
	}

	return &pb.ChunkResponse{Data: data}, nil
}

func (s *server) StoreLink(ctx context.Context, in *pb.LinkStorageRequest) (*pb.LinkStorageResponse, error) {
	hashString := in.GetHash()
	id := in.GetId()
	
	linkMap[hashString] = id
	return &pb.LinkStorageResponse{Status: "SUCCESS"}, nil
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


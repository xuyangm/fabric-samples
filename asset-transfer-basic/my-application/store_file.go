package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"flag"

	pb "github.com/xuyangm/fabric-samples/asset-transfer-basic/my-application/protos"
	"google.golang.org/grpc"
)

var (
	address = flag.String("port", ":50051", "the port of remote server")
	fn = flag.String("fn", "in", "the name of the file to be stored")
)

func main() {
	flag.Usage = func() {
		fmt.Println("Usage: ./store_file [-h] [-port string] [-fn string]")
		flag.PrintDefaults()
	}
	flag.Parse()

	// Connect to the server
	conn, err := grpc.Dial(*address, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)))
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

	// Send the chunk to the server
	response, err := client.PartitionFile(context.Background(), &pb.FilePartitionRequest{
		Data: data,
	})
	if err != nil {
		log.Fatalf("Failed to store chunk: %v", err)
	}

	fmt.Println(response.Status)
}
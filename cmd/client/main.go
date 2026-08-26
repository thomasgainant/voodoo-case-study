package main

import (
	"context"
	"log"
	"os"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/testclient"
)

func main() {
	addr := "localhost:8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	client, err := testclient.New(addr)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer client.Close()

	resp, err := client.Health(context.Background(), &pb.HealthRequest{})
	if err != nil {
		log.Fatalf("Health RPC failed: %v", err)
	}
	log.Printf("Health: status=%q", resp.Status)
}

package main

import (
	"log"
	"net"

	"voodoo-case-study/internal/server"
)

func main() {
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := server.New()
	log.Println("Starting gRPC server on :8080")
	log.Fatal(srv.Serve(lis))
}

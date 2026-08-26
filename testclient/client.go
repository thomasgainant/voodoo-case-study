package testclient

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "voodoo-case-study/gen/voodoo/v1"
)

// Client wraps the generated gRPC client for use in tests.
type Client struct {
	pb.VoodooServiceClient
	conn *grpc.ClientConn
}

func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		VoodooServiceClient: pb.NewVoodooServiceClient(conn),
		conn:                conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

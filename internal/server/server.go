package server

import (
	"net"

	"google.golang.org/grpc"

	pb "voodoo-case-study/gen/voodoo/v1"
	"voodoo-case-study/internal/router"
)

type Server struct {
	grpc *grpc.Server
}

func New() *Server {
	g := grpc.NewServer()
	pb.RegisterVoodooServiceServer(g, router.New())
	return &Server{grpc: g}
}

func (s *Server) Serve(lis net.Listener) error {
	return s.grpc.Serve(lis)
}

func (s *Server) Stop() {
	s.grpc.GracefulStop()
}

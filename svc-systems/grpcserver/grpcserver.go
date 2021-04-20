package grpcserver

import (
	"net"

	proto "github.com/ODIM-Project/ODIM/svc-systems/grpcserver/proto/system"
	"github.com/ODIM-Project/ODIM/svc-systems/grpcserver/server"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = logrus.New()

// Up is for bringing up gRPC server
func Up() {
	gs := grpc.NewServer()
	system := server.NewSystem()

	proto.RegisterSystemServer(gs, system)

	l, err := net.Listen("tcp", server.SystemGRPCServie+":8081")
	if err != nil {
		log.Fatal("While trying to get listen for the grpc, got: ", err.Error())
	}

	gs.Serve(l)

}



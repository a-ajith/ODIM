package server

import (
	"context"
	"net/http"

	"github.com/ODIM-Project/ODIM/svc-systems/grpcserver/proto/system"
	log "github.com/sirupsen/logrus"
)

const (
	// SystemGRPCServie is the service name for gRPC communication
	SystemGRPCServie = "SystemGRPCServie"
)

// System is for implemetenting SystemServer proto package interface
type System struct {
	
}

// NewSystem returns a new instance of System
func NewSystem() *System {
	return &System{}
}

// GetSystem is will provide system details
func (s *System) GetSystem(context.Context, *system.SystemRequest) (*system.SystemResponse, error) {
	log.Info("GetSystem is invoked")
	return &system.SystemResponse{
		StatusCode: http.StatusOK,
	}, nil
}

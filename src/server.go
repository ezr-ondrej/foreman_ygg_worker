package main

import (
	"context"

	pb "github.com/redhatinsights/yggdrasil/protocol"
)

// foremanServer implements the Worker gRPC service as defined by the yggdrasil
// gRPC protocol. It accepts Assignment messages, unmarshals the data into a
// string, and echoes the content back to the Dispatch service by calling the
// "Finish" method.
type foremanServer struct {
	pb.UnimplementedWorkerServer
}

// Send implements the "Send" method of the Worker gRPC service.
func (s *foremanServer) Send(ctx context.Context, d *pb.Data) (*pb.Receipt, error) {
	go startScript(ctx, d)

	// Respond to the start request that the work was accepted.
	return &pb.Receipt{}, nil
}

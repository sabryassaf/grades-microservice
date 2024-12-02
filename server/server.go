// main package to be able to run the server for now
package main

import (
	"context"
	"log"
	"net"

	gpb "github.com/BetterGR/grades-microservice/protos"
	"google.golang.org/grpc"
)

type gradesServer struct {
	// throws unimplemented error
	gpb.UnimplementedGradesServer
}

// GetStudentGrade method
func (s *gradesServer) GetStudentGrade(ctx context.Context, req *gpb.GradeRequest) (*gpb.GradeReply, error) {
	log.Printf("Recevied", req.GetStudentId())
	return &gpb.GradeReply{Grade: "100", Course: "test"}, nil
}

// main server function
func main() {
	// create a listener
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	// create a grpc server
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServer(grpcServer, &gradesServer{})

	// serve the grpc server
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

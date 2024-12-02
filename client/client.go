// main package to be able to run the client for now
package main

import (
	"context"
	"log"
	"time"

	gpb "github.com/BetterGR/grades-microservice/protos"
	"google.golang.org/grpc"
)

// define port
const (
	address = "localhost:50051"
)

// main client function
func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// create a client
	client := gpb.NewGradesClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	request := &gpb.GradeRequest{StudentId: "208247577"}
	response, err := client.GetStudentGrade(ctx, request)
	if err != nil {
		log.Fatalf("could not get grade: %v", err)
	}
	log.Printf("Grade: %s, Course: %s", response.Grade, response.Course)

}

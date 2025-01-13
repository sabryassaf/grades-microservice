// main package to be able to run the examples for now.
package main

import (
	"context"
	"log"
	"time"

	gpb "github.com/BetterGR/grades-microservice/protos"
	"google.golang.org/grpc"
)

// define Constants.
const (
	address = "localhost:50051"
)

// main examples function.
func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	// create an example.
	client := gpb.NewGradesServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	request := &gpb.GetStudentCourseGradesRequest{StudentId: "1", CourseId: "23134"}
	response, err := client.GetStudentCourseGrades(ctx, request)
	if err != nil {
		log.Fatalf("could not get grade: %v", err)
	}
	log.Printf("Student id: %s, CourseId: %s, ExamType: %s, ExamGrade: %s", response.CourseGrades.StudentId, response.CourseGrades.CourseId,
		response.CourseGrades.Exams[0].ExamType, response.CourseGrades.Exams[0].Grade)
}

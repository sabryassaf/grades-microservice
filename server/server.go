// main package to be able to run the server for now.
package main

import (
	"context"
	"log"
	"net"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

const (
	address            = "localhost:50051"
	connectionProtocol = "tcp"
)

// GradesServer is the server struct still needs to implement the GradesServiceServer interface.
type GradesServer struct {
	// throws unimplemented error.
	gpb.UnimplementedGradesServiceServer
	ms.BaseServiceServer
}

// GetStudentGrades returns all grades for a student.
func (s *GradesServer) GetStudentGrades(ctx context.Context, req *gpb.StudentId) (*gpb.StudentGrades, error) {
	logger := klog.FromContext(ctx)
	grades := &gpb.StudentGrades{
		StudentId: "123456789",
		Courses: []*gpb.StudentCourseGrades{
			{
				CourseId: "236781", Exams: []*gpb.ExamGrade{
					{Course: "236781", ExamType: "final_a", Grade: "85"},
					{Course: "236781", ExamType: "final_b", Grade: "90"},
				}, Homeworks: []*gpb.HomeworkGrade{
					{Course: "236781", HomeworkNumber: "1", Grade: "100"},
					{Course: "236781", HomeworkNumber: "2", Grade: "95"},
					{Course: "236781", HomeworkNumber: "3", Grade: "0"},
				},
			},
			{
				CourseId: "236990", Exams: []*gpb.ExamGrade{
					{Course: "236990", ExamType: "final_a", Grade: "85"},
					{Course: "236990", ExamType: "final_b", Grade: "90"},
				}, Homeworks: []*gpb.HomeworkGrade{
					{Course: "236990", HomeworkNumber: "1", Grade: "100"},
					{Course: "236990", HomeworkNumber: "2", Grade: "95"},
				},
			},
		},
	}

	logger.Info("Received request for student grades", "student_id", req.GetStudentId())

	return grades, nil
}

// GetCourseGrades returns all grades for enrolled students in a course.
func (s *GradesServer) GetCourseGrades(ctx context.Context,
	req *gpb.GetCourseGradesRequest,
) (*gpb.GetCourseGradesResponse, error) {
	logger := klog.FromContext(ctx)
	grades := []*gpb.StudentCourseGrades{
		{
			StudentId: "1", CourseId: "23134", Exams: []*gpb.ExamGrade{
				{Course: "23134", ExamType: "final_a", Grade: "85"},
				{Course: "23134", ExamType: "final_b", Grade: "90"},
			},
			Homeworks: []*gpb.HomeworkGrade{
				{Course: "23134", HomeworkNumber: "1", Grade: "100"},
				{Course: "23134", HomeworkNumber: "2", Grade: "95"},
			},
		},
		{
			StudentId: "2", CourseId: "23134", Exams: []*gpb.ExamGrade{
				{Course: "23134", ExamType: "final_a", Grade: "90"},
				{Course: "23134", ExamType: "final_b", Grade: "95"},
			},
			Homeworks: []*gpb.HomeworkGrade{
				{Course: "23134", HomeworkNumber: "1", Grade: "100"},
				{Course: "23134", HomeworkNumber: "2", Grade: "95"},
			},
		},
	}
	// log the request.
	logger.Info("Received request for course grades", "course_id", req.GetCourseId())

	return &gpb.GetCourseGradesResponse{Grades: grades}, nil
}

// GetStudentCourseGrades returns a specific student grades in specific course.
func (s *GradesServer) GetStudentCourseGrades(ctx context.Context,
	req *gpb.GetStudentCourseGradesRequest,
) (*gpb.GetStudentCourseGradesResponse, error) {
	logger := klog.FromContext(ctx)
	studentCourseGrades := &gpb.StudentCourseGrades{
		StudentId: req.GetStudentId(),
		CourseId:  req.GetCourseId(),
		Exams: []*gpb.ExamGrade{
			{Course: req.GetCourseId(), ExamType: "final_a", Grade: "85"},
			{Course: req.GetCourseId(), ExamType: "final_b", Grade: "90"},
		},
		Homeworks: []*gpb.HomeworkGrade{
			{Course: req.GetCourseId(), HomeworkNumber: "1", Grade: "100"},
		},
	}

	logger.Info("Received request for student course grades", "student_id", req.GetStudentId())

	return &gpb.GetStudentCourseGradesResponse{CourseGrades: studentCourseGrades}, nil
}

// main server function.
func main() {
	// init klog.
	klog.InitFlags(nil)
	// create a listener.
	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	// create a grpc server.
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServiceServer(grpcServer, &GradesServer{})
	klog.Info("Grades server is running on port 50051")
	// serve the grpc server.
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

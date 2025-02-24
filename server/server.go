// main package to be able to run the server for now.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
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

	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	if req.GetStudentId() == "123456789" {
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
					CourseId: "234311", Exams: []*gpb.ExamGrade{
						{Course: "234311", ExamType: "final_a", Grade: "100"},
					},
				},
			},
		}

		logger.Info("Received request for student grades", "student_id", req.GetStudentId())

		return grades, nil
	}

	grades := &gpb.StudentGrades{
		StudentId: "987654321",
		Courses: []*gpb.StudentCourseGrades{
			{
				CourseId: "236703", Exams: []*gpb.ExamGrade{
					{Course: "236703", ExamType: "final_a", Grade: "85"},
					{Course: "236703", ExamType: "final_b", Grade: "90"},
				}, Homeworks: []*gpb.HomeworkGrade{
					{Course: "236703", HomeworkNumber: "1", Grade: "100"},
					{Course: "236703", HomeworkNumber: "2", Grade: "95"},
					{Course: "236703", HomeworkNumber: "3", Grade: "0"},
				},
			},
			{
				CourseId: "234311", Exams: []*gpb.ExamGrade{
					{Course: "234311", ExamType: "final_a", Grade: "99"},
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

	_, err := s.VerifyToken(ctx, req.GetToken())
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

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

	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

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

func initGradesMicroserviceServer() (*GradesServer, error) {
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	return &GradesServer{
		BaseServiceServer:                base,
		UnimplementedGradesServiceServer: gpb.UnimplementedGradesServiceServer{},
	}, nil
}

// main server function.
func main() {
	// Initialize the server
	server, err := initGradesMicroserviceServer()
	if err != nil {
		klog.Fatalf("Failed to initialize server: %v", err)
	}

	// init klog.
	klog.InitFlags(nil)
	// create a listener.
	address := "localhost:" + os.Getenv("GRPC_PORT")

	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		klog.Error("Failed to listen", "error", err)
	}

	// create a grpc server.
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServiceServer(grpcServer, server)
	klog.Info("Grades server is running on port " + os.Getenv("GRPC_PORT"))
	// serve the grpc server.
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

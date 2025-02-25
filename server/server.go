package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	connectionProtocol = "tcp"
	logLevelDebug      = 5
)

// GradesServer is the server struct still needs to implement the GradesServiceServer interface.
type GradesServer struct {
	// throws unimplemented error.
	gpb.UnimplementedGradesServiceServer
	ms.BaseServiceServer
	db *Database
}

func initGradesMicroserviceServer(ctx context.Context) (*GradesServer, error) {
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	database, err := InitializeDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &GradesServer{
		BaseServiceServer:                base,
		UnimplementedGradesServiceServer: gpb.UnimplementedGradesServiceServer{},
		db:                               database,
	}, nil
}

// GetCourseGrades returns all students grades for a specific course for a specific semester.
func (s *GradesServer) GetCourseGrades(ctx context.Context,
	req *gpb.GetCourseGradesRequest,
) (*gpb.GetCourseGradesResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for course grades", "course_id", req.GetCourseId(),
		"semester", req.GetSemester())

	// get course grades.
	grades, err := s.db.GetCourseGrades(ctx, req.GetCourseId(), req.GetSemester().String())
	if err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	// create response.
	gradesResponse := []*gpb.SingleGrade{}
	for _, grade := range grades {
		gradesResponse = append(gradesResponse, &gpb.SingleGrade{
			GradeId:    grade.GradesID,
			StudentId:  grade.StudentID,
			CourseId:   grade.CourseID,
			Semester:   req.GetSemester(),
			GradeType:  grade.GradeType,
			ItemId:     grade.ItemID,
			GradeValue: grade.GradeValue,
			GradedBy:   grade.GradedBy,
			Comments:   grade.Comments,
		})
	}

	return &gpb.GetCourseGradesResponse{Grades: gradesResponse}, nil
}

// GetStudentCourseGrades returns all grades for a specific student in a specific course for a specific semester.
func (s *GradesServer) GetStudentCourseGrades(ctx context.Context,
	req *gpb.GetStudentCourseGradesRequest,
) (*gpb.GetStudentCourseGradesResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for student course grades", "course_id", req.GetCourseId(),
		"semester", req.GetSemester(), "student_id", req.GetStudentId())

	// get student course grades.
	grades, err := s.db.GetStudentCourseGrades(ctx, req.GetCourseId(), req.GetSemester().String(), req.GetStudentId())
	if err != nil {
		return nil, fmt.Errorf("failed to get student course grades: %w", err)
	}

	// create response.
	gradesResponse := []*gpb.SingleGrade{}
	for _, grade := range grades {
		gradesResponse = append(gradesResponse, &gpb.SingleGrade{
			GradeId:    grade.GradesID,
			StudentId:  grade.StudentID,
			CourseId:   grade.CourseID,
			Semester:   req.GetSemester(),
			GradeType:  grade.GradeType,
			ItemId:     grade.ItemID,
			GradeValue: grade.GradeValue,
			GradedBy:   grade.GradedBy,
			Comments:   grade.Comments,
		})
	}

	return &gpb.GetStudentCourseGradesResponse{Grades: gradesResponse}, nil
}

// AddSingleGrade adds a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) AddSingleGrade(ctx context.Context,
	req *gpb.AddSingleGradeRequest,
) (*gpb.AddSingleGradeResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for add single grade", "course_id", req.GetCourseId(),
		"semester", req.GetSemester(), "student_id", req.GetStudentId())

	// create grade.
	grade := &Grades{
		StudentID:  req.GetStudentId(),
		CourseID:   req.GetCourseId(),
		Semester:   req.GetSemester().String(),
		GradeType:  req.GetGradeType(),
		ItemID:     req.GetItemId(),
		GradeValue: req.GetGradeValue(),
		GradedBy:   req.GetGradedBy(),
		Comments:   req.GetComments(),
		GradedAt:   time.Now(),
		UpdatedAt:  time.Now(),
	}

	// add grade.
	if err := s.db.AddSingleGrade(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to add single grade: %w", err)
	}

	return &gpb.AddSingleGradeResponse{}, nil
}

// UpdateSingleGrade updates a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) UpdateSingleGrade(ctx context.Context,
	req *gpb.UpdateSingleGradeRequest,
) (*gpb.UpdateSingleGradeResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for update single grade", "course_id", req.GetCourseId(),
		"semester", req.GetSemester(), "student_id", req.GetStudentId())

	// create grade.
	grade := &Grades{
		GradesID:   req.GetGradeId(),
		StudentID:  req.GetStudentId(),
		CourseID:   req.GetCourseId(),
		Semester:   req.GetSemester().String(),
		GradeType:  req.GetGradeType(),
		ItemID:     req.GetItemId(),
		GradeValue: req.GetGradeValue(),
		GradedBy:   req.GetGradedBy(),
		Comments:   req.GetComments(),
		UpdatedAt:  time.Now(),
	}

	// update grade.
	if err := s.db.UpdateSingleGrade(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to update single grade: %w", err)
	}

	return &gpb.UpdateSingleGradeResponse{}, nil
}

// RemoveSingleGrade removes a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) RemoveSingleGrade(ctx context.Context,
	req *gpb.RemoveSingleGradeRequest,
) (*gpb.RemoveSingleGradeResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request to remove a single grade", "course_id", req.GetCourseId(),
		"semester", req.GetSemester(), "student_id", req.GetStudentId(), "grade_id", req.GetGradeId())

	grade := &Grades{
		GradesID:  req.GetGradeId(),
		StudentID: req.GetStudentId(),
		CourseID:  req.GetCourseId(),
		Semester:  req.GetSemester().String(),
	}

	if err := s.db.RemoveSingleGrade(ctx, grade); err != nil {
		return nil, fmt.Errorf("failed to remove single grade: %w", err)
	}

	return &gpb.RemoveSingleGradeResponse{}, nil
}

// main server function.
func main() {
	// Initialize the server
	server, err := initGradesMicroserviceServer(context.Background())
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

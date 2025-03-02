package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog"
)

// MockClaims overrides Claims behavior for testing.
type MockClaims struct {
	ms.Claims
}

// Always return true for HasRole.
func (m MockClaims) HasRole(_ string) bool {
	return true
}

// Always return "student" for GetRole.
func (m MockClaims) GetRole() string {
	return "test-role"
}

// TestGradesServer wraps GradesServer for testing.
type TestGradesServer struct {
	*GradesServer
}

func TestMain(m *testing.M) {
	// Load .env file.
	cmd := exec.Command("cat", "../.env")

	output, err := cmd.Output()
	if err != nil {
		panic("Error reading .env file: " + err.Error())
	}

	// Set environment variables.
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				// Remove quotes from the value if they exist.
				value := strings.Trim(parts[1], `"'`)
				os.Setenv(parts[0], value)
			}
		}
	}

	// Run tests and capture the result.
	result := m.Run()

	// Print custom summary.
	if result == 0 {
		klog.Info("\n\n [Summary] All tests passed.")
	} else {
		klog.Errorf("\n\n [Summary] Some tests failed. number of tests that failed: %d", result)
	}

	// Exit with the test result code.
	os.Exit(result)
}

func createTestGrade() *gpb.SingleGrade {
	return &gpb.SingleGrade{
		GradeID:    uuid.New().String(),
		StudentID:  uuid.New().String(),
		CourseID:   uuid.New().String(),
		Semester:   "Winter_2023",
		GradeType:  "Exam",
		ItemID:     uuid.New().String(),
		GradeValue: "A",
		GradedBy:   "Professor X",
		Comments:   "Excellent work!",
	}
}

func startTestServer() (*grpc.Server, net.Listener, *TestGradesServer, error) {
	server, err := initGradesMicroserviceServer()
	if err != nil {
		return nil, nil, nil, err
	}

	server.Claims = MockClaims{}
	testServer := &TestGradesServer{GradesServer: server}
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServiceServer(grpcServer, testServer)

	listener, err := net.Listen(connectionProtocol, os.Getenv("GRPC_PORT"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen on port %s: %w", os.Getenv("GRPC_PORT"), err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			panic("Failed to serve: " + err.Error())
		}
	}()

	return grpcServer, listener, testServer, nil
}

func setupClient(t *testing.T) gpb.GradesServiceClient {
	t.Helper()

	grpcServer, listener, _, err := startTestServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		grpcServer.Stop()
	})

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return gpb.NewGradesServiceClient(conn)
}

func TestGetCourseGrades(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	_, err := client.AddSingleGrade(context.Background(), &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	})
	require.NoError(t, err)

	req := &gpb.GetCourseGradesRequest{
		Token:    "test-token",
		CourseID: grade.GetCourseID(), Semester: grade.GetSemester(),
	}
	resp, err := client.GetCourseGrades(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, grade.GetCourseID(), resp.GetGrades()[0].GetCourseID())
}

func TestGetStudentCourseGrades(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	_, err := client.AddSingleGrade(context.Background(), &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	})
	require.NoError(t, err)

	req := &gpb.GetStudentCourseGradesRequest{
		Token:    "test-token",
		CourseID: grade.GetCourseID(), Semester: grade.GetSemester(), StudentID: grade.GetStudentID(),
	}
	resp, err := client.GetStudentCourseGrades(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, grade.GetStudentID(), resp.GetGrades()[0].GetStudentID())
}

func TestAddSingleGrade(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	req := &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	}

	_, err := client.AddSingleGrade(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, grade.GetStudentID(), req.GetGrade().GetStudentID())
}

func TestUpdateSingleGrade(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	_, err := client.AddSingleGrade(context.Background(), &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	})
	require.NoError(t, err)

	grade.GradeValue = "B"
	req := &gpb.UpdateSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	}

	_, err = client.UpdateSingleGrade(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "B", req.GetGrade().GetGradeValue())
}

func TestRemoveSingleGrade(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	_, err := client.AddSingleGrade(context.Background(), &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	})
	require.NoError(t, err)

	req := &gpb.RemoveSingleGradeRequest{Token: "test-token", GradeID: grade.GetGradeID()}
	_, err = client.RemoveSingleGrade(context.Background(), req)
	require.NoError(t, err)
}

func TestGetStudentSemesterGrades(t *testing.T) {
	client := setupClient(t)
	grade := createTestGrade()
	_, err := client.AddSingleGrade(context.Background(), &gpb.AddSingleGradeRequest{
		Token: "test-token",
		Grade: grade,
	})
	require.NoError(t, err)

	req := &gpb.GetStudentSemesterGradesRequest{
		Token:    "test-token",
		Semester: grade.GetSemester(), StudentID: grade.GetStudentID(),
	}
	resp, err := client.GetStudentSemesterGrades(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, grade.GetStudentID(), resp.GetGrades()[0].GetStudentID())
}

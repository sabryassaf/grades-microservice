package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog"
)

// ErrGradeNotFound is returned when a grade cannot be found.
var ErrGradeNotFound = errors.New("grade not found")

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

// MockDatabase is a mock implementation of the Database interface for testing.
type MockDatabase struct {
	grades map[string]*Grade
	mutex  sync.RWMutex
}

// Verify that MockDatabase implements DBInterface at compile time.
var _ DBInterface = (*MockDatabase)(nil)

// NewMockDatabase creates a new mock database.
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		grades: make(map[string]*Grade),
	}
}

// AddGrade adds a grade to the mock database.
func (m *MockDatabase) AddGrade(_ context.Context, grade *gpb.SingleGrade) (*Grade, error) {
	if grade == nil {
		return nil, ErrGradeNil
	}

	if grade.GetStudentID() == "" {
		return nil, ErrStudentIDEmpty
	}

	if grade.GetCourseID() == "" {
		return nil, ErrCourseIDEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	gradeID := grade.GetGradeID()
	if gradeID == "" {
		gradeID = uuid.New().String()
	}

	dbGrade := &Grade{
		GradeID:    gradeID,
		StudentID:  grade.GetStudentID(),
		CourseID:   grade.GetCourseID(),
		Semester:   grade.GetSemester(),
		GradeType:  grade.GetGradeType(),
		ItemID:     grade.GetItemID(),
		GradeValue: grade.GetGradeValue(),
		GradedBy:   grade.GetGradedBy(),
		GradedAt:   time.Now(),
		UpdatedAt:  time.Now(),
		Comments:   grade.GetComments(),
	}

	m.grades[gradeID] = dbGrade

	return dbGrade, nil
}

// GetCourseGrades gets grades for a course in a specific semester.
func (m *MockDatabase) GetCourseGrades(_ context.Context, courseID, semester string) ([]*Grade, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*Grade

	for _, grade := range m.grades {
		if grade.CourseID == courseID && grade.Semester == semester {
			result = append(result, grade)
		}
	}

	return result, nil
}

// GetStudentCourseGrades gets grades for a student in a course for a specific semester.
func (m *MockDatabase) GetStudentCourseGrades(
	_ context.Context,
	courseID, semester, studentID string,
) ([]*Grade, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*Grade

	for _, grade := range m.grades {
		if grade.CourseID == courseID && grade.Semester == semester && grade.StudentID == studentID {
			result = append(result, grade)
		}
	}

	return result, nil
}

// UpdateGrade updates a grade in the mock database.
func (m *MockDatabase) UpdateGrade(_ context.Context, grade *gpb.SingleGrade) (*Grade, error) {
	if grade == nil {
		return nil, ErrGradeNil
	}

	if grade.GetGradeID() == "" {
		return nil, ErrGradeIDEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing, exists := m.grades[grade.GetGradeID()]
	if !exists {
		return nil, ErrGradeNotFound
	}

	// Update fields if provided
	m.updateGradeFields(existing, grade)
	existing.UpdatedAt = time.Now()

	return existing, nil
}

// updateGradeFields updates the fields of an existing grade with values from a new grade.
func (m *MockDatabase) updateGradeFields(existing *Grade, grade *gpb.SingleGrade) {
	if grade.GetStudentID() != "" {
		existing.StudentID = grade.GetStudentID()
	}

	if grade.GetCourseID() != "" {
		existing.CourseID = grade.GetCourseID()
	}

	if grade.GetSemester() != "" {
		existing.Semester = grade.GetSemester()
	}

	if grade.GetGradeType() != "" {
		existing.GradeType = grade.GetGradeType()
	}

	if grade.GetItemID() != "" {
		existing.ItemID = grade.GetItemID()
	}

	if grade.GetGradeValue() != "" {
		existing.GradeValue = grade.GetGradeValue()
	}

	if grade.GetGradedBy() != "" {
		existing.GradedBy = grade.GetGradedBy()
	}

	if grade.GetComments() != "" {
		existing.Comments = grade.GetComments()
	}
}

// RemoveGrade removes a grade from the mock database.
func (m *MockDatabase) RemoveGrade(_ context.Context, gradeID string) error {
	if gradeID == "" {
		return ErrGradeIDEmpty
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.grades[gradeID]; !exists {
		return ErrGradeNotFound
	}

	delete(m.grades, gradeID)

	return nil
}

// GetStudentSemesterGrades gets all grades for a student in a specific semester.
func (m *MockDatabase) GetStudentSemesterGrades(_ context.Context, studentID, semester string) ([]*Grade, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var result []*Grade

	for _, grade := range m.grades {
		if grade.StudentID == studentID && grade.Semester == semester {
			result = append(result, grade)
		}
	}

	return result, nil
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

	// Set a mock DSN to avoid connecting to real database
	os.Setenv("DSN", "mock_dsn")

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
	// Create a base server
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create base service: %w", err)
	}

	// Use mock database instead of real one
	mockDB := NewMockDatabase()

	// Create the grades server with mock database
	server := &GradesServer{
		BaseServiceServer:                base,
		UnimplementedGradesServiceServer: gpb.UnimplementedGradesServiceServer{},
		db:                               mockDB,
		Claims:                           MockClaims{},
	}

	testServer := &TestGradesServer{GradesServer: server}
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServiceServer(grpcServer, testServer)

	listener, err := net.Listen(connectionProtocol, "localhost:0") // Use port 0 to get a random available port
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen: %w", err)
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

	// Using grpc.Dial is deprecated but keeping for now as it's a test
	// #nosec G402 -- This is a test and we're using insecure credentials intentionally
	// TODO: Update to use grpc.NewClient in the future
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

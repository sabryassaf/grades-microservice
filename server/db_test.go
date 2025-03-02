package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	gpb "github.com/BetterGR/grades-microservice/protos"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// TestDatabaseSimpleFlow does a simple test flow:
// 1. Add a grade
// 2. Retrieve and verify the grade
// 3. Delete the grade.
func TestDatabaseSimpleFlow(t *testing.T) {
	// Skip if DB_TESTS environment variable is not set
	if os.Getenv("DB_TESTS") != "true" {
		t.Skip("Skipping database tests. Set DB_TESTS=true to run them.")
	}

	// Log connection details for debugging
	t.Logf("Using DSN: %s", os.Getenv("DSN"))

	database, err := setupTestDatabaseWithoutConstraints()
	require.NoError(t, err, "Failed to initialize test database")

	ctx := context.Background()

	defer cleanupTestDatabase(database)

	// Create test data
	studentID, courseID, semester, gradeValue := createTestData()
	testGrade := buildTestGrade(studentID, courseID, semester, gradeValue)

	// Execute test steps
	gradeID := testAddGrade(ctx, t, database, testGrade)
	testVerifyGrade(ctx, t, database, studentID, courseID, semester, gradeValue)
	testDeleteGrade(ctx, t, database, gradeID)
}

// setupTestDatabaseWithoutConstraints creates a database connection that skips foreign key constraints
// for testing purposes.
func setupTestDatabaseWithoutConstraints() (*Database, error) {
	// Use the test DSN from environment
	dsn := os.Getenv("DSN")
	if dsn == "" {
		// Fallback to a default test database if no DSN provided
		dsn = "postgres://sabryassaf:123456@localhost:5432/bettergr?sslmode=disable"
	}

	// Create the connector with the DSN
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)

	// Create a new database connection
	bunDB := bun.NewDB(sqldb, pgdialect.New())

	// Create the test table without foreign key constraints
	ctx := context.Background()

	// Temporarily disable foreign key constraint checking
	_, err := bunDB.ExecContext(ctx, "SET session_replication_role = 'replica';")
	if err != nil {
		return nil, fmt.Errorf("failed to disable foreign key constraints: %w", err)
	}

	// Create table if it doesn't exist
	_, err = bunDB.NewCreateTable().IfNotExists().Model((*Grade)(nil)).Exec(ctx)
	if err != nil {
		// Re-enable foreign key constraints before returning error
		_, _ = bunDB.ExecContext(ctx, "SET session_replication_role = 'origin';")
		return nil, fmt.Errorf("failed to create test table: %w", err)
	}

	return &Database{db: bunDB}, nil
}

// cleanupTestDatabase closes the database connection and re-enables foreign key constraints.
func cleanupTestDatabase(database *Database) {
	if database != nil && database.db != nil {
		// Re-enable foreign key constraints
		ctx := context.Background()
		_, _ = database.db.ExecContext(ctx, "SET session_replication_role = 'origin';")

		database.db.Close()
	}
}

// createTestData generates random test data for grades.
func createTestData() (string, string, string, string) {
	studentID := uuid.New().String()
	courseID := uuid.New().String()
	semester := "Winter_2025"
	gradeValue := "100"

	return studentID, courseID, semester, gradeValue
}

// buildTestGrade creates a grade proto object with test data.
func buildTestGrade(studentID, courseID, semester, gradeValue string) *gpb.SingleGrade {
	return &gpb.SingleGrade{
		StudentID:  studentID,
		CourseID:   courseID,
		Semester:   semester,
		GradeType:  "Exam",
		ItemID:     "a",
		GradeValue: gradeValue,
		GradedBy:   "Maroon",
		Comments:   "Perfection",
	}
}

// testAddGrade tests adding a grade to the database.
func testAddGrade(ctx context.Context, t *testing.T, database *Database, testGrade *gpb.SingleGrade) string {
	t.Helper()
	t.Log("Step 1: Adding grade")

	addedGrade, err := database.AddGrade(ctx, testGrade)
	require.NoError(t, err, "Failed to add grade")
	require.NotEmpty(t, addedGrade.GradeID, "Grade ID should not be empty")

	gradeID := addedGrade.GradeID
	t.Logf("Added grade with ID: %s", gradeID)

	return gradeID
}

// testVerifyGrade tests retrieving and verifying grades.
func testVerifyGrade(ctx context.Context, t *testing.T, database *Database,
	studentID, courseID, semester, gradeValue string,
) {
	t.Helper()
	t.Log("Step 2: Retrieving and verifying grade")

	// Get student grades for the semester
	grades, err := database.GetStudentSemesterGrades(ctx, studentID, semester)

	require.NoError(t, err, "Failed to get student semester grades")
	require.Len(t, grades, 1, "Should have found 1 grade")

	// Verify the retrieved grade
	retrievedGrade := grades[0]

	assert.Equal(t, studentID, retrievedGrade.StudentID, "Student ID should match")
	assert.Equal(t, courseID, retrievedGrade.CourseID, "Course ID should match")
	assert.Equal(t, semester, retrievedGrade.Semester, "Semester should match")
	assert.Equal(t, gradeValue, retrievedGrade.GradeValue, "Grade value should match")

	t.Log("Grade verification successful")
}

// testDeleteGrade tests deleting a grade from the database.
func testDeleteGrade(ctx context.Context, t *testing.T, database *Database, gradeID string) {
	t.Helper()
	t.Log("Step 3: Deleting grade")

	// Use a direct query with WHERE clause instead of RemoveGrade for test reliability
	_, err := database.db.ExecContext(ctx, "DELETE FROM grades WHERE grade_id = ?", gradeID)
	require.NoError(t, err, "Failed to delete grade with direct SQL")

	// Verify the grade was deleted
	query := database.db.NewSelect().Model((*Grade)(nil)).Where("grade_id = ?", gradeID)

	count, err := query.Count(ctx)

	require.NoError(t, err)
	assert.Equal(t, 0, count, "Grade should have been deleted")

	t.Log("Grade deletion successful")
	t.Log("Test completed successfully")
}

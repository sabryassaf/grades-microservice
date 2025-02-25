package main

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	// Run tests.
	os.Exit(m.Run())
}

func createTestGrade() *Grades {
	return &Grades{
		StudentID:  uuid.New().String(),
		CourseID:   uuid.New().String(),
		Semester:   "Fall 2023",
		GradeType:  "Exam",
		ItemID:     uuid.New().String(),
		GradeValue: "A",
		GradedBy:   "Test Professor",
		GradedAt:   time.Now(),
		UpdatedAt:  time.Now(),
		Comments:   "Test Comment",
	}
}

func TestAddAndGetGrade(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Test adding a grade.
	grade := createTestGrade()
	err = db.AddSingleGrade(ctx, grade)
	assert.NoError(t, err)
	assert.NotEmpty(t, grade.GradesID)

	// Test getting the grade.
	grades, err := db.GetStudentCourseGrades(ctx, grade.CourseID, grade.Semester, grade.StudentID)
	assert.NoError(t, err)
	assert.Len(t, grades, 1)
	assert.Equal(t, grade.GradeValue, grades[0].GradeValue)
	assert.Equal(t, grade.StudentID, grades[0].StudentID)
	assert.Equal(t, grade.CourseID, grades[0].CourseID)

	// Cleanup - remove the test grade.
	err = db.RemoveSingleGrade(ctx, grade)
	assert.NoError(t, err)
}

func TestUpdateGrade(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Add a grade.
	grade := createTestGrade()
	err = db.AddSingleGrade(ctx, grade)
	assert.NoError(t, err)

	// Update the grade.
	grade.GradeValue = "B+"
	grade.Comments = "Updated Comment"
	err = db.UpdateSingleGrade(ctx, grade)
	assert.NoError(t, err)

	// Verify the update.
	grades, err := db.GetStudentCourseGrades(ctx, grade.CourseID, grade.Semester, grade.StudentID)
	assert.NoError(t, err)
	assert.Len(t, grades, 1)
	assert.Equal(t, "B+", grades[0].GradeValue)
	assert.Equal(t, "Updated Comment", grades[0].Comments)

	// Cleanup.
	err = db.RemoveSingleGrade(ctx, grade)
	assert.NoError(t, err)
}

func TestGetCourseGrades(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	courseID := uuid.New().String()
	semester := "Fall 2023"
	var testGrades []*Grades

	// Add multiple grades for the same course.
	for i := 0; i < 3; i++ {
		grade := createTestGrade()
		grade.CourseID = courseID
		grade.Semester = semester
		err := db.AddSingleGrade(ctx, grade)
		assert.NoError(t, err)
		testGrades = append(testGrades, grade)
	}

	// Get all grades for the course.
	grades, err := db.GetCourseGrades(ctx, courseID, semester)
	assert.NoError(t, err)
	assert.Len(t, grades, 3)

	// Cleanup
	for _, grade := range testGrades {
		err = db.RemoveSingleGrade(ctx, grade)
		assert.NoError(t, err)
	}
}

func TestGetNonExistentGrades(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Test getting non-existent grades.
	grades, err := db.GetStudentCourseGrades(ctx, "non-existent-course", "Fall 2023", "non-existent-student")
	assert.NoError(t, err)
	assert.Len(t, grades, 0)
}

func TestUpdateNonExistentGrade(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Try to update a non-existent grade.
	grade := createTestGrade()
	grade.GradesID = uuid.New().String()
	err = db.UpdateSingleGrade(ctx, grade)
	assert.Error(t, err)
}

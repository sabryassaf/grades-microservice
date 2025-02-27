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
		Semester:   "Winter_2025",
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
	t.Parallel()

	ctx := context.Background()
	dataBase, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, dataBase)

	// Test adding a grade.
	grade := createTestGrade()
	err = dataBase.AddSingleGrade(ctx, grade)
	require.NoError(t, err)
	assert.NotEmpty(t, grade.GradesID)

	// Test getting the grade.
	grades, err := dataBase.GetStudentCourseGrades(ctx, grade.CourseID, grade.Semester, grade.StudentID)
	require.NoError(t, err)
	assert.Len(t, grades, 1)
	assert.Equal(t, grade.GradeValue, grades[0].GradeValue)
	assert.Equal(t, grade.StudentID, grades[0].StudentID)
	assert.Equal(t, grade.CourseID, grades[0].CourseID)

	// Cleanup - remove the test grade.
	err = dataBase.RemoveSingleGrade(ctx, grade)
	assert.NoError(t, err)
}

func TestUpdateGrade(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataBase, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, dataBase)

	// Add a grade.
	grade := createTestGrade()
	err = dataBase.AddSingleGrade(ctx, grade)
	require.NoError(t, err)

	// Update the grade.
	grade.GradeValue = "B+"
	grade.Comments = "Updated Comment"
	err = dataBase.UpdateSingleGrade(ctx, grade)
	require.NoError(t, err)

	// Verify the update.
	grades, err := dataBase.GetStudentCourseGrades(ctx, grade.CourseID, grade.Semester, grade.StudentID)
	require.NoError(t, err)
	assert.Len(t, grades, 1)
	assert.Equal(t, "B+", grades[0].GradeValue)
	assert.Equal(t, "Updated Comment", grades[0].Comments)

	// Cleanup.
	err = dataBase.RemoveSingleGrade(ctx, grade)
	require.NoError(t, err)
}

func TestGetCourseGrades(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataBase, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, dataBase)

	courseID := uuid.New().String()
	semester := "Winter_2025"

	testGrades := make([]*Grades, 3)

	// Add multiple grades for the same course.
	for testGradeIndex := range 3 {
		grade := createTestGrade()
		grade.CourseID = courseID
		grade.Semester = semester
		err := dataBase.AddSingleGrade(ctx, grade)
		require.NoError(t, err)

		testGrades[testGradeIndex] = grade
	}

	// Get all grades for the course.
	grades, err := dataBase.GetCourseGrades(ctx, courseID, semester)
	require.NoError(t, err)
	assert.Len(t, grades, 3)

	// Cleanup
	for _, grade := range testGrades {
		err = dataBase.RemoveSingleGrade(ctx, grade)
		assert.NoError(t, err)
	}
}

func TestGetNonExistentGrades(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataBase, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, dataBase)

	// Test getting non-existent grades.
	grades, err := dataBase.GetStudentCourseGrades(ctx, "non-existent-course", "Winter_2025", "non-existent-student")
	require.NoError(t, err)
	assert.Empty(t, grades)
}

func TestUpdateNonExistentGrade(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dataBase, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, dataBase)

	// Try to update a non-existent grade.
	grade := createTestGrade()
	grade.GradesID = uuid.New().String()
	err = dataBase.UpdateSingleGrade(ctx, grade)
	require.Error(t, err)
}

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddGrades(t *testing.T) {
	ctx := context.Background()
	db, err := ConnectDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	testCases := []struct {
		name     string
		grade    *Grades
		wantErr  bool
		testDesc string
	}{
		{
			name: "Add Math Grade",
			grade: &Grades{
				StudentID:  "student123",
				CourseID:   "MATH101",
				Semester:   "Fall 2023",
				GradeType:  "Final",
				ItemID:     "FINAL_EXAM",
				GradeValue: "A",
				GradedBy:   "Prof. Smith",
				GradedAt:   time.Now(),
				UpdatedAt:  time.Now(),
				Comments:   "Excellent work",
			},
			wantErr:  false,
			testDesc: "Adding a math grade",
		},
		{
			name: "Add Physics Grade",
			grade: &Grades{
				StudentID:  "student456",
				CourseID:   "PHYS101",
				Semester:   "Fall 2023",
				GradeType:  "Midterm",
				ItemID:     "MIDTERM_1",
				GradeValue: "B+",
				GradedBy:   "Prof. Johnson",
				GradedAt:   time.Now(),
				UpdatedAt:  time.Now(),
				Comments:   "Good understanding of concepts",
			},
			wantErr:  false,
			testDesc: "Adding a physics grade",
		},
		{
			name: "Add Chemistry Lab Grade",
			grade: &Grades{
				StudentID:  "student789",
				CourseID:   "CHEM101",
				Semester:   "Fall 2023",
				GradeType:  "Lab",
				ItemID:     "LAB_1",
				GradeValue: "A-",
				GradedBy:   "Prof. Williams",
				GradedAt:   time.Now(),
				UpdatedAt:  time.Now(),
				Comments:   "Great lab technique",
			},
			wantErr:  false,
			testDesc: "Adding a chemistry lab grade",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := db.AddSingleGrade(ctx, tc.grade)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, tc.grade.GradesID)
				t.Logf("Successfully added %s with ID: %s", tc.testDesc, tc.grade.GradesID)
			}
		})
	}
}

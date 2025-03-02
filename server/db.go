package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	gpb "github.com/BetterGR/grades-microservice/protos"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog/v2"
)

// Database represents the database connection.
type Database struct {
	db *bun.DB
}

// Verify that Database implements DBInterface at compile time.
var _ DBInterface = (*Database)(nil)

var (
	ErrGradeNil       = errors.New("grade is nil")
	ErrStudentIDEmpty = errors.New("student ID is empty")
	ErrCourseIDEmpty  = errors.New("course ID is empty")
	ErrGradeIDEmpty   = errors.New("grade ID is empty")
)

// InitializeDatabase ensures that the database exists and initializes the schema.
func InitializeDatabase() (*Database, error) {
	createDatabaseIfNotExists()

	database, err := ConnectDB()
	if err != nil {
		return nil, err
	}

	if err := database.createSchemaIfNotExists(context.Background()); err != nil {
		klog.Fatalf("Failed to create schema: %v", err)
	}

	return database, nil
}

// createDatabaseIfNotExists ensures the database exists.
func createDatabaseIfNotExists() {
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))

	sqldb := sql.OpenDB(connector)
	defer sqldb.Close()

	ctx := context.Background()
	dbName := os.Getenv("DP_NAME")
	query := "SELECT 1 FROM pg_database WHERE datname = $1;"

	var exists int

	err := sqldb.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		klog.Fatalf("Failed to check db existence: %v", err)
	}

	if errors.Is(err, sql.ErrNoRows) {
		if _, err = sqldb.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s;", dbName)); err != nil {
			klog.Fatalf("Failed to create database: %v", err)
		}

		klog.V(logLevelDebug).Infof("Database %s created successfully.", dbName)
	} else {
		klog.V(logLevelDebug).Infof("Database %s already exists.", dbName)
	}
}

// ConnectDB connects to the database.
func ConnectDB() (*Database, error) {
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	database := bun.NewDB(sqldb, pgdialect.New())

	// Test the connection.
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	klog.V(logLevelDebug).Info("Connected to PostgreSQL database.")

	return &Database{db: database}, nil
}

// createSchemaIfNotExists creates the database schema if it doesn't exist.
func (d *Database) createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Grade)(nil),
	}

	for _, model := range models {
		if _, err := d.db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	klog.V(logLevelDebug).Info("Database schema initialized.")

	return nil
}

// Grade represents the grades table.
type Grade struct {
	GradeID    string    `bun:"grade_id,unique,pk,default:uuid_generate_v4()"`
	StudentID  string    `bun:"student_id,notnull"`
	CourseID   string    `bun:"course_id,notnull"`
	Semester   string    `bun:"semester,notnull"`
	GradeType  string    `bun:"grade_type,notnull"`
	ItemID     string    `bun:"item_id"`
	GradeValue string    `bun:"grade_value,notnull"`
	GradedBy   string    `bun:"graded_by"`
	GradedAt   time.Time `bun:"graded_at,default:current_timestamp"`
	UpdatedAt  time.Time `bun:"updated_at,default:current_timestamp"`
	Comments   string    `bun:"comments"`
}

// AddGrade adds a grade to the database.
func (d *Database) AddGrade(ctx context.Context, grade *gpb.SingleGrade) (*Grade, error) {
	if grade == nil {
		return nil, fmt.Errorf("%w", ErrGradeNil)
	}

	newGrade := &Grade{
		StudentID:  grade.GetStudentID(),
		CourseID:   grade.GetCourseID(),
		Semester:   grade.GetSemester(),
		GradeType:  grade.GetGradeType(),
		ItemID:     grade.GetItemID(),
		GradeValue: grade.GetGradeValue(),
		GradedBy:   grade.GetGradedBy(),
		Comments:   grade.GetComments(),
	}

	if _, err := d.db.NewInsert().Model(newGrade).Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to add grade: %w", err)
	}

	return newGrade, nil
}

// GetCourseGrades retrieves all grades for a course.
func (d *Database) GetCourseGrades(ctx context.Context, courseID, semester string) ([]*Grade, error) {
	if courseID == "" {
		return nil, fmt.Errorf("%w", ErrCourseIDEmpty)
	}

	var grades []*Grade
	if err := d.db.NewSelect().Model(&grades).Where("course_id = ? AND semester = ?",
		courseID, semester).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	return grades, nil
}

// GetStudentCourseGrades retrieves all grades for a student in a course.
func (d *Database) GetStudentCourseGrades(ctx context.Context,
	courseID, semester, studentID string,
) ([]*Grade, error) {
	if studentID == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	var grades []*Grade
	if err := d.db.NewSelect().Model(&grades).Where("course_id = ? AND semester = ? AND student_id = ?",
		courseID, semester, studentID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get student course grades: %w", err)
	}

	return grades, nil
}

// UpdateGrade updates a grade in the database.
func (d *Database) UpdateGrade(ctx context.Context, grade *gpb.SingleGrade) (*Grade, error) {
	if grade == nil {
		return nil, fmt.Errorf("%w", ErrGradeNil)
	}

	if grade.GetGradeID() == "" {
		return nil, fmt.Errorf("%w", ErrGradeIDEmpty)
	}

	// Get the grade from the database.
	existingGrade := &Grade{GradeID: grade.GetGradeID()}
	if err := d.db.NewSelect().Model(existingGrade).WherePK().Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get grade: %w", err)
	}

	// Update the fields.
	updateField := func(field *string, newValue string) {
		if newValue != "" {
			*field = newValue
		}
	}

	updateField(&existingGrade.StudentID, grade.GetStudentID())
	updateField(&existingGrade.CourseID, grade.GetCourseID())
	updateField(&existingGrade.Semester, grade.GetSemester())
	updateField(&existingGrade.GradeType, grade.GetGradeType())
	updateField(&existingGrade.ItemID, grade.GetItemID())
	updateField(&existingGrade.GradeValue, grade.GetGradeValue())
	updateField(&existingGrade.GradedBy, grade.GetGradedBy())
	updateField(&existingGrade.Comments, grade.GetComments())

	if _, err := d.db.NewUpdate().Model(existingGrade).WherePK().Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to update grade: %w", err)
	}

	return existingGrade, nil
}

// DeleteGrade deletes a grade from the database.
func (d *Database) RemoveGrade(ctx context.Context, gradeID string) error {
	if gradeID == "" {
		return fmt.Errorf("%w", ErrGradeIDEmpty)
	}

	grade := &Grade{GradeID: gradeID}
	if _, err := d.db.NewDelete().Model(grade).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete grade: %w", err)
	}

	return nil
}

// GetStudentSemesterGrades retrieves all grades for a student in a semester.
func (d *Database) GetStudentSemesterGrades(ctx context.Context, studentID, semester string) ([]*Grade, error) {
	if studentID == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	var grades []*Grade
	if err := d.db.NewSelect().Model(&grades).Where("student_id = ? AND semester = ?",
		studentID, semester).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get student semester grades: %w", err)
	}

	return grades, nil
}

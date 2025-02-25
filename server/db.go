package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog/v2"
)

// Database represents the database connection.
type Database struct {
	db *bun.DB
}

// InitializeDatabase ensures that the database exists and initializes the schema.
func InitializeDatabase(ctx context.Context) (*Database, error) {
	createDatabaseIfNotExists(ctx) // Create bettergr if it does not exist.

	database, err := ConnectDB(ctx) // Connect to bettergr.
	if err != nil {
		return nil, err
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Database initialized successfully.")

	if err := database.createSchemaIfNotExists(ctx); err != nil {
		klog.Fatalf("Failed to create schema: %v", err)
	}

	logger.V(logLevelDebug).Info("Schema created successfully.")

	return database, nil
}

// createDatabaseIfNotExists checks if the database exists and creates it if not.
func createDatabaseIfNotExists(ctx context.Context) {
	// Connect to the PostgreSQL server (not the bettergr itself yet).
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	logger := klog.FromContext(ctx)

	defer sqldb.Close()

	// Check if the database exists.
	query := `
		SELECT 1 FROM pg_database WHERE datname = 'bettergrdatabase';
	`

	var exists int

	err := sqldb.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		klog.Fatalf("Failed to check if database exists: %v", err)
	}

	// If the database does not exist, create it.
	if errors.Is(err, sql.ErrNoRows) {
		if _, err := sqldb.ExecContext(ctx, `CREATE DATABASE bettergrdatabase;`); err != nil {
			klog.Fatalf("Failed to create database: %v", err)
		}

		logger.V(logLevelDebug).Info("Database bettergrdatabase created successfully.")
	} else {
		logger.V(logLevelDebug).Info("Database bettergrdatabase already exists.")
	}
}

// ConnectDB initializes the PostgreSQL database connection.
func ConnectDB(ctx context.Context) (*Database, error) {
	dsn := os.Getenv("DSN")
	connector := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	sqldb := sql.OpenDB(connector)
	database := bun.NewDB(sqldb, pgdialect.New())
	logger := klog.FromContext(ctx)
	// Test the connection.
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}

	logger.V(logLevelDebug).Info("Connected to PostgreSQL database.")

	return &Database{db: database}, nil
}

// createSchemaIfNotExists creates the schema if it does not exist.
func (db *Database) createSchemaIfNotExists(ctx context.Context) error {
	models := []interface{}{
		(*Grades)(nil),
	}

	for _, model := range models {
		if _, err := db.db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	klog.Info("Database schema initialized.")

	return nil
}

// Grades represents the grades table.
type Grades struct {
	GradesID   string    `bun:"grades_id,pk,autoincrement"`
	StudentID  string    `bun:"student_id"`
	CourseID   string    `bun:"course_id"`
	Semester   string    `bun:"semester"`
	GradeType  string    `bun:"grade_type"`
	ItemID     string    `bun:"item_id"`
	GradeValue string    `bun:"grade_value"`
	GradedBy   string    `bun:"graded_by"`
	GradedAt   time.Time `bun:"graded_at"`
	UpdatedAt  time.Time `bun:"updated_at"`
	Comments   string    `bun:"comments"`
}

// GetStudentCourseGrades returns all grades for a specific course for a specific semester for a specific student.
func (db *Database) GetStudentCourseGrades(ctx context.Context, courseID string,
	semester string, studentID string,
) ([]*Grades, error) {
	grades := []*Grades{}

	if err := db.db.NewSelect().Model(&grades).Where("course_id = ?", courseID).Where("semester = ?",
		semester).Where("student_id = ?", studentID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	return grades, nil
}

// AddSingleGrade adds a grade for a specific course for a specific semester for a specific student.
func (db *Database) AddSingleGrade(ctx context.Context, grade *Grades) error {
	// create unique grade id.
	grade.GradesID = uuid.New().String()
	if _, err := db.db.NewInsert().Model(grade).Exec(ctx); err != nil {
		return fmt.Errorf("failed to add grade: %w", err)
	}

	return nil
}

// UpdateSingleGrade updates a grade for a specific course for a specific semester for a specific student.
func (db *Database) UpdateSingleGrade(ctx context.Context, grade *Grades) error {
	// find unique grade.
	existingGrade := &Grades{}
	if err := db.db.NewSelect().Model(existingGrade).Where("grades_id = ?",
		grade.GradesID).Where("student_id = ?", grade.StudentID).Where("course_id = ?",
		grade.CourseID).Where("semester = ?", grade.Semester).Scan(ctx); err != nil {
		return fmt.Errorf("failed to find grade: %w", err)
	}

	if _, err := db.db.NewUpdate().Model(grade).Where("grades_id = ?",
		existingGrade.GradesID).Where("student_id = ?", existingGrade.StudentID).Where("course_id = ?",
		existingGrade.CourseID).Where("semester = ?", existingGrade.Semester).Exec(ctx); err != nil {
		return fmt.Errorf("failed to update grade: %w", err)
	}

	return nil
}

// RemoveSingleGrade removes a grade for a specific course for a specific semester for a specific student.
func (db *Database) RemoveSingleGrade(ctx context.Context, grade *Grades) error {
	if _, err := db.db.NewDelete().Model(grade).Where("grades_id = ?",
		grade.GradesID).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete grade: %w", err)
	}

	return nil
}

// GetCourseGrades returns all students grades for a specific course for a specific semester.
func (db *Database) GetCourseGrades(ctx context.Context, courseID string, semester string) ([]*Grades, error) {
	grades := []*Grades{}

	if err := db.db.NewSelect().Model(&grades).Where(
		"course_id = ?", courseID).Where("semester = ?", semester).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	return grades, nil
}

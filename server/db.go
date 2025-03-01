package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	spb "github.com/BetterGR/students-microservice/protos"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"k8s.io/klog/v2"
)

// Database represents the database connection.
type Database struct {
	db *bun.DB
}

var (
	ErrStudentNil      = errors.New("student is nil")
	ErrStudentIDEmpty  = errors.New("student ID is empty")
	ErrStudentNotFound = errors.New("student not found")
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
		(*Student)(nil),
	}

	for _, model := range models {
		if _, err := d.db.NewCreateTable().IfNotExists().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	klog.V(logLevelDebug).Info("Database schema initialized.")

	return nil
}

// Student represents the database schema for students.
type Student struct {
	StudentID   string    `bun:"student_id,unique,pk,notnull"`
	FirstName   string    `bun:"first_name,notnull"`
	LastName    string    `bun:"last_name,notnull"`
	Email       string    `bun:"email,unique,notnull"`
	PhoneNumber string    `bun:"phone_number,unique,notnull"`
	CreatedAt   time.Time `bun:"created_at,default:current_timestamp"`
	UpdatedAt   time.Time `bun:"updated_at,default:current_timestamp"`
}

// AddStudent adds a new student to the database.
func (d *Database) AddStudent(ctx context.Context, student *spb.Student) (*Student, error) {
	if student == nil {
		return nil, fmt.Errorf("%w", ErrStudentNil)
	}

	newStudent := &Student{
		StudentID:   student.GetStudentID(),
		FirstName:   student.GetFirstName(),
		LastName:    student.GetLastName(),
		Email:       student.GetEmail(),
		PhoneNumber: student.GetPhoneNumber(),
	}

	_, err := d.db.NewInsert().Model(newStudent).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to add student: %w", err)
	}

	return newStudent, nil
}

// GetStudent retrieves a student by ID from the database.
func (d *Database) GetStudent(ctx context.Context, studentID string) (*Student, error) {
	if studentID == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	student := new(Student)
	if err := d.db.NewSelect().Model(student).Where("student_id = ?", studentID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	return student, nil
}

// UpdateStudent updates an existing student in the database.
func (d *Database) UpdateStudent(ctx context.Context, student *spb.Student) (*Student, error) {
	if student == nil {
		return nil, fmt.Errorf("%w", ErrStudentNil)
	}

	if student.GetStudentID() == "" {
		return nil, fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	// get the existing student.
	existingStudent, err := d.GetStudent(ctx, student.GetStudentID())
	if err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	// Update the fields.
	updateField := func(field *string, newValue string) {
		if newValue != "" {
			*field = newValue
		}
	}

	updateField(&existingStudent.FirstName, student.GetFirstName())
	updateField(&existingStudent.LastName, student.GetLastName())
	updateField(&existingStudent.Email, student.GetEmail())
	updateField(&existingStudent.PhoneNumber, student.GetPhoneNumber())

	if _, err := d.db.NewUpdate().Model(existingStudent).WherePK().Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to update grade: %w", err)
	}

	return existingStudent, nil
}

// DeleteStudent removes a student from the database.
func (d *Database) DeleteStudent(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w", ErrStudentIDEmpty)
	}

	res, err := d.db.NewDelete().Model((*Student)(nil)).Where("student_id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	if num, _ := res.RowsAffected(); num == 0 {
		return fmt.Errorf("%w", ErrStudentNotFound)
	}

	return nil
}

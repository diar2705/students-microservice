package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

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

		klog.Infof("Database %s created successfully.", dbName)
	} else {
		klog.Infof("Database %s already exists.", dbName)
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

	klog.Info("Connected to PostgreSQL database.")

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

	klog.Info("Database schema initialized.")

	return nil
}

// Student represents the database schema for students.
type Student struct {
	UniqueID    string   `bun:",pk,default:gen_random_uuid()"`
	ID          string   `bun:"id,unique,notnull"`
	FirstName   string   `bun:"first_name,notnull"`
	LastName    string   `bun:"last_name,notnull"`
	Email       string   `bun:"email,unique,notnull"`
	PhoneNumber string   `bun:"phone_number,unique,notnull"`
	Courses     []string `bun:",array"`
}

// AddStudent adds a new student to the database.
func (d *Database) AddStudent(ctx context.Context, student *spb.Student) error {
	_, err := d.db.NewInsert().Model(&Student{
		ID:          student.GetId(),
		FirstName:   student.GetFirstName(),
		LastName:    student.GetSecondName(),
		Email:       student.GetEmail(),
		PhoneNumber: student.GetPhoneNumber(),
		Courses:     student.GetCourses(),
	}).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to add student: %w", err)
	}

	return nil
}

// GetStudent retrieves a student by ID from the database.
func (d *Database) GetStudent(ctx context.Context, id string) (*spb.Student, error) {
	student := new(Student)

	if err := d.db.NewSelect().Model(student).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get student: %w", err)
	}

	return &spb.Student{
		Id:          student.ID,
		FirstName:   student.FirstName,
		SecondName:  student.LastName,
		Email:       student.Email,
		PhoneNumber: student.PhoneNumber,
		Courses:     student.Courses,
	}, nil
}

// UpdateStudent updates an existing student in the database.
func (d *Database) UpdateStudent(ctx context.Context, student *spb.Student) error {
	_, err := d.db.NewUpdate().Model(&Student{
		ID:          student.GetId(),
		FirstName:   student.GetFirstName(),
		LastName:    student.GetSecondName(),
		Email:       student.GetEmail(),
		PhoneNumber: student.GetPhoneNumber(),
		Courses:     student.GetCourses(),
	}).Where("id = ?", student.GetId()).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update student: %w", err)
	}

	return nil
}

// DeleteStudent removes a student from the database.
func (d *Database) DeleteStudent(ctx context.Context, id string) error {
	_, err := d.db.NewDelete().Model((*Student)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete student: %w", err)
	}

	return nil
}

// GetStudentCourses retrieves the courses of a student by ID.
func (d *Database) GetStudentCourses(ctx context.Context, id string) ([]string, error) {
	student := new(Student)

	if err := d.db.NewSelect().Model(student).Where("id = ?", id).Scan(ctx); err != nil {
		return nil, fmt.Errorf("failed to get student courses: %w", err)
	}

	return student.Courses, nil
}

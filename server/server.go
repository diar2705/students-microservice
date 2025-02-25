// main package to be able to run the StudentsServer for now
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	spb "github.com/BetterGR/students-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	// define address.
	connectionProtocol = "tcp"
	// Debugging logs.
	logLevelDebug = 5
)

// StudentsServer is an implementation of GRPC Students microservice.
type StudentsServer struct {
	ms.BaseServiceServer
	db *Database
	// throws unimplemented error
	spb.UnimplementedStudentsServiceServer
}

// initStudentsMicroserviceServer initializes the StudentsServer.
func initStudentsMicroserviceServer() (*StudentsServer, error) {
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	database, err := InitializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &StudentsServer{
		BaseServiceServer:                  base,
		db:                                 database,
		UnimplementedStudentsServiceServer: spb.UnimplementedStudentsServiceServer{},
	}, nil
}

// GetStudent search for the Student that corresponds to the given id and returns them.
func (s *StudentsServer) GetStudent(ctx context.Context,
	req *spb.GetStudentRequest,
) (*spb.GetStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received GetStudent request", "studentId", req.GetId())

	student, err := s.db.GetStudent(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "student not found: %v", err)
	}

	return &spb.GetStudentResponse{
		Student: student,
	}, nil
}

// CreateStudent creates a new Student with the given details and returns them.
func (s *StudentsServer) CreateStudent(ctx context.Context,
	req *spb.CreateStudentRequest,
) (*spb.CreateStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received CreateStudent request",
		"firstName", req.GetStudent().GetFirstName(), "secondName", req.GetStudent().GetSecondName())

	if err := s.db.AddStudent(ctx, req.GetStudent()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create student: %v", err)
	}

	return &spb.CreateStudentResponse{Student: req.GetStudent()}, nil
}

// UpdateStudent updates the given Student and returns them after the update.
func (s *StudentsServer) UpdateStudent(ctx context.Context,
	req *spb.UpdateStudentRequest,
) (*spb.UpdateStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received UpdateStudent request",
		"firstName", req.GetStudent().GetFirstName(), "secondName", req.GetStudent().GetSecondName())

	if err := s.db.UpdateStudent(ctx, req.GetStudent()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update student: %v", err)
	}

	return &spb.UpdateStudentResponse{Student: req.GetStudent()}, nil
}

// GetStudentCourses searches the courses that the Student is enrolled in during the given semester and returns them.
func (s *StudentsServer) GetStudentCourses(ctx context.Context,
	req *spb.GetStudentCoursesRequest,
) (*spb.GetStudentCoursesResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.Info("Received GetStudentCourses request",
		"ID", req.GetId(),
		"semester", req.GetSemester())

	courses, err := s.db.GetStudentCourses(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "courses not found: %v", err)
	}

	return &spb.GetStudentCoursesResponse{
		Courses: courses,
	}, nil
}

// DeleteStudent deletes the Student from the system.
func (s *StudentsServer) DeleteStudent(ctx context.Context,
	req *spb.DeleteStudentRequest,
) (*spb.DeleteStudentResponse, error) {
	if _, err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received DeleteStudent request", "studentId", req.GetStudent().GetId())

	if err := s.db.DeleteStudent(ctx, req.GetStudent().GetId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete student: %v", err)
	}

	logger.Info("Deleted", "studentId", req.GetStudent().GetId())

	return &spb.DeleteStudentResponse{}, nil
}

// main StudentsServer function.
func main() {
	// init klog
	klog.InitFlags(nil)
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		klog.Fatalf("Error loading .env file")
	}

	// init the StudentsServer
	server, err := initStudentsMicroserviceServer()
	if err != nil {
		klog.Fatalf("Failed to init StudentsServer: %v", err)
	}

	// create a listener on port 'address'
	address := os.Getenv("GRPC_PORT")

	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		klog.Fatalf("Failed to listen: %v", err)
	}

	klog.Info("Starting StudentsServer on port: ", address)
	// create a grpc StudentsServer
	grpcServer := grpc.NewServer()
	spb.RegisterStudentsServiceServer(grpcServer, server)

	// serve the grpc StudentsServer
	if err := grpcServer.Serve(lis); err != nil {
		klog.Fatalf("Failed to serve: %v", err)
	}
}

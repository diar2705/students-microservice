package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"

	spb "github.com/BetterGR/students-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/klog"
)

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

// TestStudentsServer wraps StudentsServer for testing.
type TestStudentsServer struct {
	*StudentsServer
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

	// Run tests and capture the result.
	result := m.Run()

	if result == 0 {
		klog.Info("\n\n [Summary] All tests passed.")
	} else {
		klog.Errorf("\n\n [Summary] Some tests failed. number of tests that failed: %d", result)
	}

	// Exit with the test result code.
	os.Exit(result)
}

func createTestStudent() *spb.Student {
	return &spb.Student{
		StudentID:   uuid.New().String(),
		FirstName:   "John",
		SecondName:  "Doe",
		Email:       "john.doe@example.com",
		PhoneNumber: "1234567890",
	}
}

func startTestServer() (*grpc.Server, net.Listener, *TestStudentsServer, error) {
	server, err := initStudentsMicroserviceServer()
	if err != nil {
		return nil, nil, nil, err
	}

	server.Claims = MockClaims{}
	testServer := &TestStudentsServer{StudentsServer: server}
	grpcServer := grpc.NewServer()
	spb.RegisterStudentsServiceServer(grpcServer, testServer)

	listener, err := net.Listen(connectionProtocol, os.Getenv("GRPC_PORT"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen on port %s: %w", os.Getenv("GRPC_PORT"), err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			panic("Failed to serve: " + err.Error())
		}
	}()

	return grpcServer, listener, testServer, nil
}

func setupClient(t *testing.T) spb.StudentsServiceClient {
	t.Helper()

	grpcServer, listener, _, err := startTestServer()
	require.NoError(t, err)
	t.Cleanup(func() {
		grpcServer.Stop()
	})

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return spb.NewStudentsServiceClient(conn)
}

func TestGetStudentFound(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	_, err := client.CreateStudent(t.Context(), &spb.CreateStudentRequest{Student: student, Token: "test-token"})
	require.NoError(t, err)

	req := &spb.GetStudentRequest{StudentID: student.GetStudentID(), Token: "test-token"}
	resp, err := client.GetStudent(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, student.GetStudentID(), resp.GetStudent().GetStudentID())

	// Cleanup.
	_, _ = client.DeleteStudent(t.Context(), &spb.DeleteStudentRequest{Student: student, Token: "test-token"})
}

func TestGetStudentNotFound(t *testing.T) {
	client := setupClient(t)
	req := &spb.GetStudentRequest{StudentID: "non-existent-id", Token: "test-token"}

	_, err := client.GetStudent(t.Context(), req)
	assert.Error(t, err)
}

func TestCreateStudentSuccessful(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	req := &spb.CreateStudentRequest{Student: student, Token: "test-token"}

	_, err := client.CreateStudent(t.Context(), req)
	require.NoError(t, err)

	// Cleanup.
	_, _ = client.DeleteStudent(t.Context(), &spb.DeleteStudentRequest{Student: student, Token: "test-token"})
}

func TestCreateStudentFailureOnDuplicate(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	_, err := client.CreateStudent(t.Context(), &spb.CreateStudentRequest{Student: student, Token: "test-token"})
	require.NoError(t, err)

	req := &spb.CreateStudentRequest{Student: student, Token: "test-token"}
	_, err = client.CreateStudent(t.Context(), req)
	require.Error(t, err)

	// Cleanup.
	_, _ = client.DeleteStudent(t.Context(), &spb.DeleteStudentRequest{Student: student, Token: "test-token"})
}

func TestUpdateStudentSuccessful(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	_, err := client.CreateStudent(t.Context(), &spb.CreateStudentRequest{Student: student, Token: "test-token"})
	require.NoError(t, err)

	// Modify student.
	student.FirstName = "UpdatedName"
	req := &spb.UpdateStudentRequest{Student: student, Token: "test-token"}

	_, err = client.UpdateStudent(t.Context(), req)
	require.NoError(t, err)

	// Cleanup.
	_, _ = client.DeleteStudent(t.Context(), &spb.DeleteStudentRequest{Student: student, Token: "test-token"})
}

func TestUpdateStudentFailureForNonExistentStudent(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	student.StudentID = "non-existent-id"
	req := &spb.UpdateStudentRequest{Student: student, Token: "test-token"}

	_, err := client.UpdateStudent(t.Context(), req)
	assert.Error(t, err)
}

func TestDeleteStudentSuccessful(t *testing.T) {
	client := setupClient(t)
	student := createTestStudent()
	_, err := client.CreateStudent(t.Context(), &spb.CreateStudentRequest{Student: student, Token: "test-token"})
	require.NoError(t, err)

	req := &spb.DeleteStudentRequest{Student: student, Token: "test-token"}
	_, err = client.DeleteStudent(t.Context(), req)
	assert.NoError(t, err)
}

func TestDeleteStudentFailureForNonExistentStudent(t *testing.T) {
	client := setupClient(t)
	req := &spb.DeleteStudentRequest{Student: &spb.Student{StudentID: "non-existent-id"}, Token: "test-token"}

	_, err := client.DeleteStudent(t.Context(), req)
	assert.Error(t, err)
}

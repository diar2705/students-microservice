# Students Microservice

This repository provides a gRPC service for managing student records. The service is implemented in Go and uses Protobuf to define message types and service methods.

## Setup

### 1. Setup Prerequisites

Make sure you have the following installed:

- Go (latest stable version recommended)
- Protobuf Compiler (protoc)
- Docker (if using Docker deployment)
- PostgreSQL database (ensure PostgreSQL is installed and running)

### 2. Clone the Repository

```bash
git clone https://github.com/BetterGR/students-microservice.git
```

### 3. Set Up Environment Variables

Create a `.env` file in the root directory with the following environment variables:

```.env
GRPC_PORT=localhost:50052
AUTH_ISSUER=http://auth.BetterGR.org
DSN=postgres://postgres:bettergr2425@localhost:5432/bettergr?sslmode=disable
DB_NAME=bettergr
```

#### Testing Environment Variables

For test environments, you can set a separate database connection string using:

```.env
DSN_TEST=postgres://postgres:password@localhost:5432/test_db?sslmode=disable
DP_NAME=students_test
```

The service is designed to look for `DSN_TEST` first and use it for tests if available. This allows you to use a separate test database while keeping your production database (`DSN`) untouched.

### 4. Configure MicroService Library

This repository depends on the TekClinic/MicroService-Lib library for authentication and environment variable management. Proper configuration of the required environment variables from TekClinic/MicroService-Lib is essential. Refer to its documentation for proper setup.

### 5. Start the gRPC Server

To start the server, open the terminal in the students-microservice directory and run the following:

```bash
go mod init github.com/BetterGR/students-microservice
make run
```

### 6. Testing

#### Local Testing

To run unit tests locally:

```bash
make test
```

Make sure you have PostgreSQL running locally. You can use either:

- The regular `DSN` environment variable
- A test-specific `DSN_TEST` environment variable (which will take precedence)

#### CI Testing

This repository is configured to run tests automatically in GitHub Actions. The workflow:

1. Sets up Go
2. Installs a PostgreSQL instance
3. Runs the tests with the appropriate environment variables, using a dedicated `DSN_TEST` variable for the test database connection

The CI configuration can be found in `.github/workflows/ci.yml`.

### 7. Makefile Help

For more available commands and their descriptions, run:

```bash
make help
```

## License

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.

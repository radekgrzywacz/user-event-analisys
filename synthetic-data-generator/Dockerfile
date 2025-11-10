# Start from the official Golang image
FROM golang:1.24.2

WORKDIR /app

# Copy Go module files
COPY synthetic-data-generator/go.mod synthetic-data-generator/go.sum ./
# Copy contracts module before downloading dependencies
COPY contracts /contracts

# Download dependencies
RUN go mod download

# Copy the full source code
COPY synthetic-data-generator/ ./

# Build the Go app using the correct main file
RUN go build -o synthetic-data-generator ./cmd/main

# Run the binary
CMD ["./synthetic-data-generator"]

# Start from the official Golang image
FROM golang:1.24.2

# Set the working directory inside the container
WORKDIR /app

# # Copy Go module files
COPY http-ingestor/go.mod http-ingestor/go.sum ./

COPY contracts /contracts

# Download dependencies
RUN go mod download

# Copy the full source code
COPY http-ingestor/ ./

# Build the Go app using the correct main file
RUN go build -o http-ingestor ./cmd/main

# Run the binary
CMD ["./http-ingestor"]

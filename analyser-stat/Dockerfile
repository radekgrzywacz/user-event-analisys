# Start from the official Golang image
FROM golang:1.24.2

WORKDIR /app

# Copy Go module files
COPY analyser-stat/go.mod analyser-stat/go.sum ./

# Copy contracts module before downloading dependencies (needed by replace ../contracts)
COPY contracts /contracts

# Download dependencies
RUN go mod download

# Copy the full source code
COPY analyser-stat/ ./

# Build the Go app using the correct main file
RUN go build -o analyser-stat ./cmd/main

# Run the binary
CMD ["./analyser-stat"]

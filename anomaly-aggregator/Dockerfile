# Start from the official Golang image
FROM golang:1.24.2

WORKDIR /app

# Copy Go module files
COPY anomaly-aggregator/go.mod anomaly-aggregator/go.sum ./

# Copy contracts module before downloading dependencies (needed by replace ../contracts)
COPY contracts /contracts

# Download dependencies
RUN go mod download

# Copy the full source code
COPY anomaly-aggregator/ ./

# Build the Go app using the correct main file
RUN go build -o anomaly-aggregator ./cmd/main

# Run the binary
CMD ["./anomaly-aggregator"]

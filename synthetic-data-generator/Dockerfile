# Start from the official Golang image
FROM golang:1.24.2

# Set the working directory inside the container
WORKDIR /synthetic-data-generator

# Copy Go modules manifests
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of your app
COPY . .

# Build the Go app
RUN go build -o synthetic-data-generator .

# Run the binary
CMD ["./synthetic-data-generator"]

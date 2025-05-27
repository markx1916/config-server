# Stage 1: Build the Go application
FROM golang:1.21-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download
RUN go mod verify

# Copy the source code into the container
COPY . .

# Build the Go app
# -ldflags="-w -s" reduces the size of the binary by removing debug information
# CGO_ENABLED=0 disables Cgo to build a statically-linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s" -o nacos-config-tool-api cmd/server/main.go

# Stage 2: Create the production image
# Use a minimal base image like alpine
FROM alpine:latest

# Add ca-certificates for HTTPS calls if needed by the application
RUN apk --no-cache add ca-certificates

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/nacos-config-tool-api .

# Copy the example configuration file. The actual config.yaml should be mounted as a volume.
COPY config.yaml.example /app/config.yaml.example

# Expose port 8080 to the outside world (or the port configured)
EXPOSE 8080

# Command to run the executable
# The config file path can be overridden by environment variables or command-line arguments if the app supports it.
# By default, the app should look for config.yaml in its working directory or predefined paths.
ENTRYPOINT ["./nacos-config-tool-api"]

# First stage: build the Go application
FROM golang:1.23-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN GOOS=linux GOARCH=amd64 go build -o main .

# Second stage: create a minimal image
FROM scratch

# Copy the binary from the builder stage
COPY --from=builder /app/main /main

# Expose port 8080 to the outside world
EXPOSE 5050

# Command to run the executable
CMD ["/main"]
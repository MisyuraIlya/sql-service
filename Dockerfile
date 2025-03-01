FROM golang:1.24.0-alpine

WORKDIR /app

# Copy the go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application from the cmd folder
RUN go build -o main ./cmd

# Expose the port the app runs on
EXPOSE 8080

# Run the application
CMD ["./main"]

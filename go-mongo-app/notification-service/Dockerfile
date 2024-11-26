FROM golang:1.20 AS builder

WORKDIR /app

# Copy go.mod and go.sum for dependency management
COPY go.mod go.sum ./

# Clear Go module cache
RUN rm -rf /go/pkg/mod && go clean -modcache

# Download dependencies
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

# Use a minimal base image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
RUN chmod +x ./app
EXPOSE 8080
ENV ENABLE_BOOTSTRAP=true
CMD ["./app"]

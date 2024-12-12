# Start from Go image
FROM golang:1.20

# Set the working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the application
COPY . .

# Build the application
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]

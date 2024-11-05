# Use the official Go image
FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . . 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
RUN chmod +x ./app
EXPOSE 8080
ENV ENABLE_BOOTSTRAP=true
CMD ["./app"]

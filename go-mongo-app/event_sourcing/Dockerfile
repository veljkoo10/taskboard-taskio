# Stage 1: Build stage
FROM golang:1.20 AS builder

WORKDIR /app

# Kopiramo mod fajlove i resetujemo keš
COPY go.mod go.sum ./
RUN rm -rf /go/pkg/mod && go clean -modcache
RUN go mod download

# Kopiramo ostatak aplikacije i gradimo binarnu
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

# Stage 2: Minimal runtime stage
FROM alpine:latest

WORKDIR /root/

# Kopiramo izgrađenu binarnu datoteku
COPY --from=builder /app/app .

# Dodajemo permisije za izvršavanje
RUN chmod +x ./app

# Izlažemo port aplikacije
EXPOSE 8080

# Dodajemo varijable za konfiguraciju
ENV ENABLE_BOOTSTRAP=true

# Dodajemo logger za sve outpute
CMD ["./app"]

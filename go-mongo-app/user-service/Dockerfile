# Use the official Go image
FROM golang:1.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Kopiraj blacklist.txt u /app/service unutar kontejnera
COPY ./service/blacklist.txt ./service/blacklist.txt
COPY . . 
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app .

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/app .
COPY --from=builder /app/service/blacklist.txt /root/service/blacklist.txt

# Testiraj da li je fajl na pravom mestu
RUN ls -alh /root/service
RUN chmod +x ./app
EXPOSE 8080
ENV ENABLE_BOOTSTRAP=true
CMD ["./app"]

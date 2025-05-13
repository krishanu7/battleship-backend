# Build Stage
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . .

# Build the main backend binary
RUN CGO_ENABLED=0 GOOS=linux go build -o battleship-backend .

# Build the matchmaker binary
RUN CGO_ENABLED=0 GOOS=linux go build -o matchmaker ./cmd/matchmaker

# Backend runtime stage
FROM alpine:latest AS backend
WORKDIR /app
COPY --from=builder /app/battleship-backend .
EXPOSE 8080
CMD ["./battleship-backend"]

# Matchmaker runtime stage
FROM alpine:latest AS matchmaker
WORKDIR /app
COPY --from=builder /app/matchmaker .
CMD ["./matchmaker"]


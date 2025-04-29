FROM golang:1.24

WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod go.sum ./
RUN go mod download

# Copy rest of the source code
COPY . .

# Build the Go app
RUN go build -o battleship-backend .

# Expose the app port
EXPOSE 8080

# Run the binary
CMD ["./battleship-backend"]

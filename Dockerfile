FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy only go.mod first
COPY go.mod ./

# Download dependencies (this creates go.sum inside the container if needed)
RUN go mod download

# Copy the rest of the files
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o forgejo-mcp-server main.go

# Final lightweight image
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/forgejo-mcp-server .

EXPOSE 8080
CMD ["./forgejo-mcp-server"]

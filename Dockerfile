FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod init forgejo-mcp-server
RUN go mod tidy
RUN go build -o forgejo-mcp-server main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/forgejo-mcp-server .
EXPOSE 8080
CMD ["./forgejo-mcp-server"]

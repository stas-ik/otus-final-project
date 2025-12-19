# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o gomigrator ./cmd/gomigrator/main.go

# Stage 2: Final image
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gomigrator .

ENTRYPOINT ["./gomigrator"]

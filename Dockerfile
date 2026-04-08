# Dockerfile untuk traffic-extractor (Go/Gin)
# File ini di-copy ke folder extractor/ oleh setup.sh saat pertama kali setup

FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o extractor .

# ----
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/extractor .

EXPOSE 8081

CMD ["./extractor"]

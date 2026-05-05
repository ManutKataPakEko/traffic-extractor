# Request Extractor

A small Go service that receives HTTP traffic, sanitizes the request body, forwards the request to a gRPC ML service for prediction, and writes the enriched result to `requests.log`.

## What it does

For each incoming HTTP request, the service:

1. Collects request metadata
   - timestamp
   - IP
   - method
   - path
   - query string
   - headers
   - body
2. Sends the data to the ML gRPC service through `Predict`
3. Adds the returned `prediction` to the payload
4. Writes the enriched record to `requests.log`
5. Prints the same enriched record to stdout

## Requirements

- Go 1.25 or newer
- A running gRPC ML server that exposes:
  - service: `attackdetection.AttackDetection`
  - method: `Predict`
- `protoc` only if you need to regenerate the proto files

## Configuration

Create a `.env` file in the project root.

Example:

```env
ML_RPC_ADDR=10.0.0.1:50051
```

If `.env` is missing, the app falls back to:

```env
ML_RPC_ADDR=localhost:9000
```

## Local setup

### 1. Clone or open the project

```bash
cd /home/yogarn/projects/ojs/extractor
```

### 2. Create your `.env`

Copy the example file:

```bash
cp .env.example .env
```

Then edit `.env` and set the correct ML gRPC address.

### 3. Run the service

```bash
go run main.go
```

The server listens on port `8081`.

## Build

```bash
go build ./...
```

## Docker

The project includes a simple Dockerfile.

### Build the image

```bash
docker build -t request-extractor .
```

### Run the container

```bash
docker run --rm -p 8081:8081 --env-file .env request-extractor
```

## gRPC proto

The gRPC contract is defined in:

- [proto/attack_detection.proto](proto/attack_detection.proto)

Generated Go code is stored in:

- [proto/attack_detection.pb.go](proto/attack_detection.pb.go)
- [proto/attack_detection_grpc.pb.go](proto/attack_detection_grpc.pb.go)

## Regenerate protobuf code

If you change the proto, regenerate the Go files with:

```bash
protoc --go_out=. \
  --go_opt=paths=source_relative \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative \
  proto/attack_detection.proto
```

## Output file

The service writes one JSON object per line to:

- `requests.log`

## Notes

- Sensitive body parameters such as passwords are redacted before forwarding.
- Authorization and Cookie headers are removed before the payload is sent to the ML service.

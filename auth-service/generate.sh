#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.32.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0
export PATH="$(go env GOPATH)/bin:$PATH"
mkdir -p proto
protoc -I../proto \
  --go_out=./proto --go_opt=paths=source_relative \
  --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
  ../proto/auth.proto
echo "Auth gRPC code generated successfully"

#!/bin/bash

# Generate gRPC code from proto files
protoc -I../proto \
  --go_out=./proto --go_opt=paths=source_relative \
  --go-grpc_out=./proto --go-grpc_opt=paths=source_relative \
  ../proto/auth.proto

echo "✓ Auth gRPC code generated successfully"

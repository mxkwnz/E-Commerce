$root = Split-Path -Parent $PSScriptRoot
$bashCmd = 'export PATH=/usr/local/go/bin:$PATH && apt-get update -qq && apt-get install -y -qq protobuf-compiler >/dev/null && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 && export PATH=$PATH:$(go env GOPATH)/bin && protoc -I proto -I third_party --go_out=proto --go_opt=paths=source_relative --go-grpc_out=proto --go-grpc_opt=paths=source_relative proto/order.proto'

docker run --rm -v "${root}:/work" -w /work/order-service golang:1.23-bookworm bash -lc "$bashCmd"

@echo off
protoc -I=../proto --go_out=. --go_opt=paths=source_relative ^
    --go-grpc_out=. --go-grpc_opt=paths=source_relative ^
    ../proto/product.proto

if %errorlevel% equ 0 (
    echo ✓ Product gRPC code generated successfully
) else (
    echo ✗ Error during generation
)
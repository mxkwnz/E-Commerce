#!/usr/bin/env sh
set -e
root="$(cd "$(dirname "$0")/.." && pwd)"

run() {
  name="$1"
  dir="$2"
  shift 2
  echo ""
  echo "========== $name =========="
  (cd "$root/$dir" && go test "$@" -count=1 -v)
}

run "auth-service" "auth-service" ./internal/service/ ./internal/usecase/
run "order-service" "order-service" ./internal/service/
run "product-service" "product-service" ./internal/service/
run "payment-service" "payment-service" ./internal/service/ ./internal/grpc/

echo ""
echo "All tests passed."

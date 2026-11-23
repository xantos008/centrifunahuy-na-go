#!/bin/sh
# Run the licensed, local-only Azure Service Bus comparison on its own stack.
set -eu
if [ "${ACCEPT_EULA:-}" != Y ]; then
  echo "Supply ACCEPT_EULA=Y after accepting both the Microsoft Service Bus Emulator and SQL Server Linux terms." >&2
  exit 1
fi
if [ "$#" -eq 0 ]; then set -- diff golden; fi
drain_only=false
if [ "$#" -eq 1 ] && [ "$1" = drain ]; then drain_only=true; fi
run_drain=false
for mode do
  case "$mode" in
    diff|golden) run_drain=true ;;
    record) ;;
    drain) if [ "$drain_only" != true ]; then echo "Use drain on its own" >&2; exit 1; fi ;;
    *) echo "Expected diff, record or golden, or a standalone drain" >&2; exit 1 ;;
  esac
done

cd "$(dirname "$0")"
COMPAT_COMPOSE_FILE="$PWD/compose.azure.yml"
COMPAT_AZURE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/cedrazeka-azure.XXXXXXXX")
COMPOSE_PROJECT_NAME="cedrazeka-azure-$$"
MSSQL_SA_PASSWORD="Aa1!$(openssl rand -hex 20)"
export COMPAT_COMPOSE_FILE COMPAT_AZURE_DIR COMPOSE_PROJECT_NAME MSSQL_SA_PASSWORD
export COMPAT_PROFILE=azure
compose() { docker compose -f "$COMPAT_COMPOSE_FILE" --profile azure "$@"; }
cleanup() {
  compose down --volumes
  rm -rf "$COMPAT_AZURE_DIR"
}
trap cleanup EXIT
trap 'exit 1' INT TERM HUP
cp configs/azure/entities.json "$COMPAT_AZURE_DIR/ref.json"
cp configs/azure/entities.json "$COMPAT_AZURE_DIR/sut.json"
start_emulator() {
  compose up -d --wait --wait-timeout 240 "servicebus-$1"
  endpoint=$(compose port "servicebus-$1" 5300)
  curl --fail --silent --show-error --retry 80 --retry-all-errors --retry-delay 2 \
    --max-time 3 --retry-max-time 180 "http://$endpoint/health" >/dev/null
}
check_drain() {
  endpoint=$(compose port servicebus-sut 5672)
  case "$endpoint" in
    127.0.0.1:*) port=${endpoint#127.0.0.1:} ;;
    *) echo "Expected the SUT Service Bus binding on 127.0.0.1" >&2; exit 1 ;;
  esac
  case "$port" in
    ''|*[!0-9]*) echo "Expected a numeric SUT Service Bus port" >&2; exit 1 ;;
  esac
  COMPAT_AZURE_DRAIN_CONNECTION_STRING="Endpoint=sb://127.0.0.1:$port;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=SAS_KEY_VALUE;UseDevelopmentEmulator=true;" \
    go -C ../.. test -p 1 -tags=integration,azure_emulator -timeout 2m -count=1 -v \
      -run '^TestAzureServiceBusConsumer_DrainAcksInFlight$' ./internal/consuming
}
if [ "$drain_only" = true ]; then
  start_emulator sut
  check_drain
  exit 0
fi
GOOS=linux CGO_ENABLED=0 go -C ../.. build -o "$COMPAT_AZURE_DIR/cedrazeka" .
start_emulator ref
start_emulator sut
compose up -d ref sut
COMPAT_REF_URL="http://$(compose port ref 8000)"
COMPAT_SUT_URL="http://$(compose port sut 8000)"
export COMPAT_REF_URL COMPAT_SUT_URL
for mode do
  COMPAT_MODE="$mode" COMPAT_RECORD=CS-CONS-07 go test -timeout 3m -count=1 -v -run '^TestCS_CONS_07$' ./scenarios
done
if [ "$run_drain" = true ]; then check_drain; fi

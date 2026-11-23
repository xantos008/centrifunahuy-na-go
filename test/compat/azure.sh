#!/bin/sh
# Run the licensed, local-only Azure Service Bus comparison on its own stack.
set -eu
if [ "${ACCEPT_EULA:-}" != Y ]; then
  echo "Supply ACCEPT_EULA=Y after accepting both the Microsoft Service Bus Emulator and SQL Server Linux terms." >&2
  exit 1
fi
if [ "$#" -eq 0 ]; then set -- diff golden; fi
for mode do
  case "$mode" in diff|record|golden) ;; *) echo "Expected diff, record or golden" >&2; exit 1 ;; esac
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
GOOS=linux CGO_ENABLED=0 go -C ../.. build -o "$COMPAT_AZURE_DIR/cedrazeka" .
for side in ref sut; do
  compose up -d --wait --wait-timeout 240 "servicebus-$side"
  endpoint=$(compose port "servicebus-$side" 5300)
  curl --fail --silent --show-error --retry 80 --retry-all-errors --retry-delay 2 \
    --max-time 3 --retry-max-time 180 "http://$endpoint/health" >/dev/null
done
compose up -d ref sut
COMPAT_REF_URL="http://$(compose port ref 8000)"
COMPAT_SUT_URL="http://$(compose port sut 8000)"
export COMPAT_REF_URL COMPAT_SUT_URL
for mode do
  COMPAT_MODE="$mode" COMPAT_RECORD=CS-CONS-07 go test -timeout 3m -count=1 -v -run '^TestCS_CONS_07$' ./scenarios
done

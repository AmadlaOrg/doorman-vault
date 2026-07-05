#!/bin/bash
set -euo pipefail

CONTAINER_NAME="vault-test"
VAULT_PORT=8200
VAULT_TOKEN="test-token"
VAULT_ADDR="http://127.0.0.1:${VAULT_PORT}"
BINARY="./bin/linux/amd64/doorman-vault"

cleanup() {
    echo "Cleaning up..."
    podman stop "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

# Build the binary
echo "==> Building doorman-vault..."
make build-linux

# Start Vault dev server
echo "==> Starting Vault dev server in podman..."
podman run --rm -d \
    --name "$CONTAINER_NAME" \
    -p "${VAULT_PORT}:8200" \
    -e "VAULT_DEV_ROOT_TOKEN_ID=${VAULT_TOKEN}" \
    hashicorp/vault:latest

# Wait for Vault to be ready
echo "==> Waiting for Vault to be ready..."
for i in $(seq 1 30); do
    if curl -sf "${VAULT_ADDR}/v1/sys/health" > /dev/null 2>&1; then
        echo "    Vault is ready."
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "ERROR: Vault did not become ready in time."
        exit 1
    fi
    sleep 1
done

# Write test secrets
echo "==> Writing test secrets..."
curl -sf -X POST \
    -H "X-Vault-Token: ${VAULT_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"data": {"username": "admin", "password": "s3cret"}}' \
    "${VAULT_ADDR}/v1/secret/data/myapp/config" > /dev/null

curl -sf -X POST \
    -H "X-Vault-Token: ${VAULT_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{"data": {"host": "db.example.com", "port": "5432"}}' \
    "${VAULT_ADDR}/v1/secret/data/myapp/database" > /dev/null

echo "==> Running tests..."
export VAULT_ADDR VAULT_TOKEN
PASS=0
FAIL=0

# Test 1: info command
echo -n "  Test info command... "
OUTPUT=$($BINARY info)
if echo "$OUTPUT" | grep -q '"doorman-vault"'; then
    echo "PASS"
    PASS=$((PASS + 1))
else
    echo "FAIL"
    echo "    Output: $OUTPUT"
    FAIL=$((FAIL + 1))
fi

# Test 2: get secret with bare path
echo -n "  Test get myapp/config... "
OUTPUT=$($BINARY get myapp/config)
if echo "$OUTPUT" | grep -q '"admin"' && echo "$OUTPUT" | grep -q '"s3cret"'; then
    echo "PASS"
    PASS=$((PASS + 1))
else
    echo "FAIL"
    echo "    Output: $OUTPUT"
    FAIL=$((FAIL + 1))
fi

# Test 3: get secret with secret/ prefix
echo -n "  Test get secret/myapp/database... "
OUTPUT=$($BINARY get secret/myapp/database)
if echo "$OUTPUT" | grep -q '"db.example.com"' && echo "$OUTPUT" | grep -q '"5432"'; then
    echo "PASS"
    PASS=$((PASS + 1))
else
    echo "FAIL"
    echo "    Output: $OUTPUT"
    FAIL=$((FAIL + 1))
fi

# Test 4: get secret with full KV v2 path
echo -n "  Test get secret/data/myapp/config... "
OUTPUT=$($BINARY get secret/data/myapp/config)
if echo "$OUTPUT" | grep -q '"admin"'; then
    echo "PASS"
    PASS=$((PASS + 1))
else
    echo "FAIL"
    echo "    Output: $OUTPUT"
    FAIL=$((FAIL + 1))
fi

# Test 5: get nonexistent secret
echo -n "  Test get nonexistent path... "
if $BINARY get nonexistent/path 2>/dev/null; then
    echo "FAIL (expected failure)"
    FAIL=$((FAIL + 1))
else
    echo "PASS (correctly failed)"
    PASS=$((PASS + 1))
fi

echo ""
echo "==> Results: ${PASS} passed, ${FAIL} failed"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi

echo "==> All integration tests passed."

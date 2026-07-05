# doorman-vault

HashiCorp Vault secret store plugin for [Doorman](https://github.com/AmadlaOrg/doorman).

This is a standalone CLI binary following the Amadla UNIX plugin protocol. It communicates with Vault's KV v2 HTTP API to retrieve secrets.

## Installation

```bash
make build
```

The binary is placed in `bin/<os>/<arch>/doorman-vault`. Add it to your `PATH` so Doorman can discover it.

## Usage

### Environment Variables

| Variable      | Required | Default                   | Description          |
|---------------|----------|---------------------------|----------------------|
| `VAULT_ADDR`  | No       | `http://127.0.0.1:8200`  | Vault server address |
| `VAULT_TOKEN` | Yes      |                           | Vault auth token     |

### Commands

**Plugin metadata:**

```bash
doorman-vault info
```

**Retrieve a secret:**

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=my-token

# These are equivalent:
doorman-vault get myapp/config
doorman-vault get secret/myapp/config
doorman-vault get secret/data/myapp/config
```

Output is the secret data as a JSON object on stdout.

### Exit Codes

| Code | Meaning     |
|------|-------------|
| 0    | Success     |
| 1    | Failure     |
| 2    | Usage error |

## Testing

```bash
# Unit tests
go test ./...

# Integration test (requires podman)
./test/podman-test.sh
```

## License

See repository root.

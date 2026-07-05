package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "doorman-vault")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "failed to build binary")
	return bin
}

func TestInfoCommand(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "info", "-o", "json")
	out, err := cmd.Output()
	require.NoError(t, err)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(out, &metadata))
	assert.Equal(t, "doorman-vault", metadata["name"])
	assert.Equal(t, "vault", metadata["engine"])
	assert.Equal(t, "1.0.0", metadata["version"])
	assert.Equal(t, "HashiCorp Vault secret store plugin for Doorman", metadata["description"])

	supports, ok := metadata["supports"].([]any)
	require.True(t, ok)
	assert.Contains(t, supports, "amadla.org/entity/secret@^v1.0.0")
}

func TestGetCommand_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/secret/data/myapp/config", r.URL.Path)
		assert.Equal(t, "test-token", r.Header.Get("X-Vault-Token"))

		resp := map[string]any{
			"data": map[string]any{
				"data": map[string]any{
					"username": "admin",
					"password": "s3cret",
				},
				"metadata": map[string]any{
					"version": 1,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	bin := buildBinary(t)
	cmd := exec.Command(bin, "get", "myapp/config")
	cmd.Env = append(os.Environ(),
		"VAULT_ADDR="+server.URL,
		"VAULT_TOKEN=test-token",
	)

	out, err := cmd.Output()
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, json.Unmarshal(out, &data))
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "s3cret", data["password"])
}

func TestGetCommand_MissingToken(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "get", "myapp/config")
	cmd.Env = []string{} // empty env, no VAULT_TOKEN
	err := cmd.Run()
	assert.Error(t, err)
}

func TestGetCommand_MissingKey(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "get")
	err := cmd.Run()
	assert.Error(t, err)
}

func TestVersionFlag(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "--version")
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "1.0.0")
}

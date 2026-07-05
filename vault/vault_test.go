package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSecret_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/secret/data/myapp/config", r.URL.Path)
		assert.Equal(t, "test-token", r.Header.Get("X-Vault-Token"))
		assert.Equal(t, http.MethodGet, r.Method)

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

	client := New(server.URL, "test-token")
	data, err := client.GetSecret("myapp/config")
	require.NoError(t, err)

	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "s3cret", data["password"])
}

func TestGetSecret_WithSecretPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/secret/data/myapp/db", r.URL.Path)

		resp := map[string]any{
			"data": map[string]any{
				"data": map[string]any{
					"host": "db.example.com",
				},
				"metadata": map[string]any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	data, err := client.GetSecret("secret/myapp/db")
	require.NoError(t, err)
	assert.Equal(t, "db.example.com", data["host"])
}

func TestGetSecret_WithFullKVPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/secret/data/full/path", r.URL.Path)

		resp := map[string]any{
			"data": map[string]any{
				"data": map[string]any{
					"key": "value",
				},
				"metadata": map[string]any{},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	data, err := client.GetSecret("secret/data/full/path")
	require.NoError(t, err)
	assert.Equal(t, "value", data["key"])
}

func TestGetSecret_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":["no secret at this path"]}`))
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	_, err := client.GetSecret("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestGetSecret_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":["permission denied"]}`))
	}))
	defer server.Close()

	client := New(server.URL, "bad-token")
	_, err := client.GetSecret("secret-path")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestGetSecret_EmptyData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	_, err := client.GetSecret("empty")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no secret data")
}

func TestGetSecret_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	client := New(server.URL, "tok")
	_, err := client.GetSecret("bad-json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding response")
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myapp/config", "secret/data/myapp/config"},
		{"secret/myapp/config", "secret/data/myapp/config"},
		{"secret/data/myapp/config", "secret/data/myapp/config"},
		{"/myapp/config", "secret/data/myapp/config"},
		{"/secret/myapp/config", "secret/data/myapp/config"},
		{"simple", "secret/data/simple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizePath(tt.input))
		})
	}
}

func TestNew(t *testing.T) {
	client := New("http://localhost:8200/", "my-token")
	assert.Equal(t, "http://localhost:8200", client.Addr)
	assert.Equal(t, "my-token", client.Token)
	assert.NotNil(t, client.HTTPClient)
}

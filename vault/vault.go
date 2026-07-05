package vault

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client communicates with a HashiCorp Vault server over HTTP.
type Client struct {
	Addr       string
	Token      string
	HTTPClient *http.Client
}

// New creates a Vault client with the given address and token.
func New(addr, token string) *Client {
	return &Client{
		Addr:       strings.TrimRight(addr, "/"),
		Token:      token,
		HTTPClient: http.DefaultClient,
	}
}

// vaultResponse represents the JSON structure returned by Vault KV v2.
type vaultResponse struct {
	Data *vaultData `json:"data"`
}

type vaultData struct {
	Data     map[string]any `json:"data"`
	Metadata map[string]any `json:"metadata"`
}

// GetSecret retrieves a secret from Vault KV v2 at the given path.
// If the path does not start with "secret/data/", it is automatically prefixed.
// Returns the secret data (the inner "data" field from Vault's response).
func (c *Client) GetSecret(path string) (map[string]any, error) {
	path = normalizePath(path)

	url := fmt.Sprintf("%s/v1/%s", c.Addr, path)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting secret: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault returned status %d: %s", resp.StatusCode, string(body))
	}

	var vr vaultResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if vr.Data == nil || vr.Data.Data == nil {
		return nil, fmt.Errorf("no secret data at path %q", path)
	}

	return vr.Data.Data, nil
}

// normalizePath ensures the path has the "secret/data/" prefix for KV v2.
func normalizePath(path string) string {
	path = strings.TrimLeft(path, "/")

	if strings.HasPrefix(path, "secret/data/") {
		return path
	}

	// Handle "secret/<key>" without "data/" segment.
	if strings.HasPrefix(path, "secret/") {
		return "secret/data/" + strings.TrimPrefix(path, "secret/")
	}

	return "secret/data/" + path
}

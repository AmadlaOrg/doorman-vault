package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AmadlaOrg/doorman-vault/vault"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	appName = "doorman-vault"
	version = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:     appName,
	Short:   "HashiCorp Vault secret store plugin for Doorman",
	Version: version,
}

var (
	infoOutputFlag string
	infoHeryFlag   bool

	infoCmd = &cobra.Command{
		Use:   "info",
		Short: "Show plugin metadata",
		Run: func(cmd *cobra.Command, args []string) {
			metadata := map[string]any{
				"name":        appName,
				"version":     version,
				"engine":      "vault",
				"supports":    []string{"amadla.org/entity/secret@^v1.0.0"},
				"description": "HashiCorp Vault secret store plugin for Doorman",
			}
			if err := writeInfoOutput(os.Stdout, infoOutputFlag, infoHeryFlag, metadata); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding metadata: %v\n", err)
				os.Exit(1)
			}
		},
	}
)

var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Retrieve a secret from Vault",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr := os.Getenv("VAULT_ADDR")
		if addr == "" {
			addr = "http://127.0.0.1:8200"
		}

		token := os.Getenv("VAULT_TOKEN")
		if token == "" {
			fmt.Fprintln(os.Stderr, "Error: VAULT_TOKEN environment variable is required")
			os.Exit(1)
		}

		client := vault.New(addr, token)
		data, err := client.GetSecret(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding output: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	infoCmd.Flags().StringVarP(&infoOutputFlag, "output", "o", "table", "Output format: table, json, yaml")
	infoCmd.Flags().BoolVar(&infoHeryFlag, "hery", false, "Wrap output in HERY envelope (_type, _body)")

	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(getCmd)
}

type heryEnvelope struct {
	Type string `json:"_type" yaml:"_type"`
	Body any    `json:"_body" yaml:"_body"`
}

func writeInfoOutput(w io.Writer, format string, hery bool, data map[string]any) error {
	if hery {
		return writeHeryOutput(w, format, data)
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		bytes, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(bytes))
		return err
	default:
		table := tablewriter.NewWriter(w)
		table.Header("Field", "Value")
		table.Append("Name", fmt.Sprint(data["name"]))
		table.Append("Version", fmt.Sprint(data["version"]))
		if eng, ok := data["engine"]; ok {
			table.Append("Engine", fmt.Sprint(eng))
		}
		table.Append("Description", fmt.Sprint(data["description"]))
		if supports, ok := data["supports"].([]string); ok {
			table.Append("Supports", strings.Join(supports, "\n"))
		}
		if exts, ok := data["file_extensions"].([]string); ok {
			table.Append("File Extensions", strings.Join(exts, ", "))
		}
		table.Render()
		return nil
	}
}

func writeHeryOutput(w io.Writer, format string, data map[string]any) error {
	envelope := heryEnvelope{
		Type: "amadla.org/entity/tools/info@v1.0.0",
		Body: data,
	}

	switch format {
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(envelope)
	case "table":
		fmt.Fprintf(w, "_type: %s\n\n", envelope.Type)
		table := tablewriter.NewWriter(w)
		table.Header("Field", "Value")
		table.Append("Name", fmt.Sprint(data["name"]))
		table.Append("Version", fmt.Sprint(data["version"]))
		if eng, ok := data["engine"]; ok {
			table.Append("Engine", fmt.Sprint(eng))
		}
		table.Append("Description", fmt.Sprint(data["description"]))
		if supports, ok := data["supports"].([]string); ok {
			table.Append("Supports", strings.Join(supports, "\n"))
		}
		if exts, ok := data["file_extensions"].([]string); ok {
			table.Append("File Extensions", strings.Join(exts, ", "))
		}
		table.Render()
		return nil
	default:
		bytes, err := yaml.Marshal(envelope)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, string(bytes))
		return err
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(2)
	}
}

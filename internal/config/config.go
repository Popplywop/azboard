package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type AuthMethod string

const (
	AuthPAT   AuthMethod = "pat"
	AuthAzCLI AuthMethod = "azcli"
)

type Config struct {
	AuthMethod           AuthMethod
	OrgURL               string // Full org URL, e.g. https://dev.azure.com/pdidev or https://pdidev.visualstudio.com
	Org                  string // Extracted org name, e.g. pdidev
	Project              string
	PAT                  string   // Only set when AuthMethod == "pat"
	Repos                []string // Comma-separated repo names
	WorkItemTypes        []string
	DefaultMergeStrategy string // "squash", "merge", "rebase", "semilinear"
	AreaPath             string // e.g. "PDI\Wholesale" — filters work items by area
}

// configJSON is the on-disk JSON representation of the config file.
type configJSON struct {
	AuthMethod           string   `json:"auth_method,omitempty"`
	OrgURL               string   `json:"org_url,omitempty"`
	Project              string   `json:"project,omitempty"`
	PAT                  string   `json:"pat,omitempty"`
	Repos                []string `json:"repos,omitempty"`
	WorkItemTypes        []string `json:"work_item_types,omitempty"`
	DefaultMergeStrategy string   `json:"default_merge_strategy,omitempty"`
	AreaPath             string   `json:"area_path,omitempty"`
}

// Load reads configuration from the default config file path.
func Load() (*Config, error) {
	return LoadFromFile(ConfigFilePath())
}

// readConfigJSON reads and unmarshals the config file at path.
func readConfigJSON(path string) (configJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return configJSON{}, fmt.Errorf("could not read config file: %w", err)
	}
	var raw configJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return configJSON{}, fmt.Errorf("invalid JSON in config file: %w", err)
	}
	return raw, nil
}

// LoadFromFile reads configuration from the given JSON file path.
func LoadFromFile(path string) (*Config, error) {
	raw, err := readConfigJSON(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	// Auth method
	switch strings.ToLower(raw.AuthMethod) {
	case "pat":
		cfg.AuthMethod = AuthPAT
	case "azcli", "":
		cfg.AuthMethod = AuthAzCLI
	default:
		return nil, fmt.Errorf("unknown auth method %q: must be \"pat\" or \"azcli\"", raw.AuthMethod)
	}

	// Org URL — used to derive org name and base URL
	if raw.OrgURL != "" {
		parsedOrg, parsedProject := parseOrgURL(raw.OrgURL)
		cfg.OrgURL = raw.OrgURL
		if parsedOrg != "" {
			cfg.Org = parsedOrg
		}
		if parsedProject != "" {
			cfg.Project = parsedProject
		}
	}

	// Project can also be set explicitly (overrides URL-parsed value)
	if raw.Project != "" {
		cfg.Project = raw.Project
	}

	// PAT
	if cfg.AuthMethod == AuthPAT {
		cfg.PAT = raw.PAT
		if cfg.PAT == "" {
			return nil, fmt.Errorf("PAT auth requires \"pat\" field in %s", path)
		}
	}

	// Repos
	cfg.Repos = raw.Repos

	// WorkItemTypes
	if len(raw.WorkItemTypes) > 0 {
		cfg.WorkItemTypes = raw.WorkItemTypes
	} else {
		cfg.WorkItemTypes = []string{"User Story", "Bug", "Task", "Feature", "Epic"}
	}

	// DefaultMergeStrategy
	switch raw.DefaultMergeStrategy {
	case "squash", "merge", "rebase", "semilinear":
		cfg.DefaultMergeStrategy = raw.DefaultMergeStrategy
	default:
		cfg.DefaultMergeStrategy = "squash"
	}

	// AreaPath
	cfg.AreaPath = raw.AreaPath

	// Validate
	if cfg.Org == "" {
		return nil, fmt.Errorf("organization is required: set \"org_url\" in %s", path)
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required: set \"project\" in %s", path)
	}

	return cfg, nil
}

// UpdateRepos reads the existing config, updates only the repos field, and writes back.
func UpdateRepos(repos []string) error {
	path := ConfigFilePath()

	raw, err := readConfigJSON(path)
	if err != nil {
		return err
	}

	raw.Repos = repos

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, append(out, '\n'), 0600)
}

// parseOrgURL extracts the org name and optional project from an Azure DevOps URL.
// Supports:
//
//	https://dev.azure.com/orgname
//	https://dev.azure.com/orgname/project
//	https://orgname.visualstudio.com
//	https://orgname.visualstudio.com/project
func parseOrgURL(rawURL string) (org, project string) {
	u, err := url.Parse(strings.TrimRight(rawURL, "/"))
	if err != nil {
		return "", ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")

	if strings.HasSuffix(u.Host, ".visualstudio.com") {
		org = strings.TrimSuffix(u.Host, ".visualstudio.com")
		if len(parts) > 0 && parts[0] != "" {
			project = parts[0]
		}
		return org, project
	}

	if u.Host == "dev.azure.com" {
		if len(parts) > 0 {
			org = parts[0]
		}
		if len(parts) > 1 {
			project = parts[1]
		}
		return org, project
	}

	return "", ""
}

// ConfigFilePath returns the default config file path.
func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.config/azboard/config.json"
	}
	return filepath.Join(home, ".config", "azboard", "config.json")
}

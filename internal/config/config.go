package config

import (
	"bufio"
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
	Repos                []string // AZBOARD_REPOS, comma-separated repo names
	WorkItemTypes        []string // AZBOARD_WORK_ITEM_TYPES, comma-separated
	DefaultMergeStrategy string   // AZBOARD_DEFAULT_MERGE_STRATEGY (default: "squash")
	AreaPath             string   // AZBOARD_AREA_PATH, e.g. "PDI\Wholesale" — filters work items by area
}

// Load reads configuration exclusively from ~/.config/azboard/config.env.
func Load() (*Config, error) {
	vals := loadConfigFile("")

	cfg := &Config{}

	// Auth method
	authStr := vals["AZURE_DEVOPS_AUTH_METHOD"]
	switch strings.ToLower(authStr) {
	case "pat":
		cfg.AuthMethod = AuthPAT
	case "azcli", "":
		cfg.AuthMethod = AuthAzCLI
	default:
		return nil, fmt.Errorf("unknown auth method %q: must be \"pat\" or \"azcli\"", authStr)
	}

	// Org URL — used to derive org name and base URL
	orgURL := vals["AZURE_DEVOPS_ORG_URL"]
	if orgURL != "" {
		parsedOrg, parsedProject := parseOrgURL(orgURL)
		cfg.OrgURL = orgURL
		if parsedOrg != "" {
			cfg.Org = parsedOrg
		}
		if parsedProject != "" {
			cfg.Project = parsedProject
		}
	}

	// Project can also be set explicitly (overrides URL-parsed value)
	if p := vals["AZURE_DEVOPS_DEFAULT_PROJECT"]; p != "" {
		cfg.Project = p
	}

	// PAT
	if cfg.AuthMethod == AuthPAT {
		cfg.PAT = vals["AZURE_DEVOPS_PAT"]
		if cfg.PAT == "" {
			return nil, fmt.Errorf("PAT auth requires AZURE_DEVOPS_PAT in %s", ConfigFilePath())
		}
	}

	// Repos: AZBOARD_REPOS — split on comma, trim whitespace
	if reposStr := vals["AZBOARD_REPOS"]; reposStr != "" {
		for _, r := range strings.Split(reposStr, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				cfg.Repos = append(cfg.Repos, r)
			}
		}
	}

	// WorkItemTypes: AZBOARD_WORK_ITEM_TYPES
	if wiTypesStr := vals["AZBOARD_WORK_ITEM_TYPES"]; wiTypesStr != "" {
		for _, t := range strings.Split(wiTypesStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				cfg.WorkItemTypes = append(cfg.WorkItemTypes, t)
			}
		}
	} else {
		cfg.WorkItemTypes = []string{"User Story", "Bug", "Task", "Feature", "Epic"}
	}

	// DefaultMergeStrategy: AZBOARD_DEFAULT_MERGE_STRATEGY
	switch vals["AZBOARD_DEFAULT_MERGE_STRATEGY"] {
	case "squash", "merge", "rebase", "semilinear":
		cfg.DefaultMergeStrategy = vals["AZBOARD_DEFAULT_MERGE_STRATEGY"]
	default:
		cfg.DefaultMergeStrategy = "squash"
	}

	// AreaPath: AZBOARD_AREA_PATH — optional, filters work items by area
	cfg.AreaPath = vals["AZBOARD_AREA_PATH"]

	// Validate
	if cfg.Org == "" {
		return nil, fmt.Errorf("organization is required: set AZURE_DEVOPS_ORG_URL in %s", ConfigFilePath())
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required: set AZURE_DEVOPS_DEFAULT_PROJECT in %s", ConfigFilePath())
	}

	return cfg, nil
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

// loadConfigFile reads a key=value config file from ~/.config/azboard/config.env.
func loadConfigFile(path string) map[string]string {
	vals := make(map[string]string)

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return vals
		}
		path = filepath.Join(home, ".config", "azboard", "config.env")
	}

	f, err := os.Open(path)
	if err != nil {
		return vals
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		vals[key] = value
	}

	return vals
}

// ConfigFilePath returns the default config file path.
func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.config/azboard/config.env"
	}
	return filepath.Join(home, ".config", "azboard", "config.env")
}

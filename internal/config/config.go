package config

import (
	"bufio"
	"flag"
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
	AuthMethod AuthMethod
	OrgURL     string // Full org URL, e.g. https://dev.azure.com/pdidev or https://pdidev.visualstudio.com
	Org        string // Extracted org name, e.g. pdidev
	Project    string
	PAT        string // Only set when AuthMethod == "pat"
}

func Load() (*Config, error) {
	org := flag.String("org", "", "Azure DevOps organization name (overrides config)")
	project := flag.String("project", "", "Azure DevOps project name (overrides config)")
	configPath := flag.String("config", "", "Path to config file (default: ~/.config/azboard/config.env)")
	flag.Parse()

	// Load config file values first (lowest priority)
	fileVals := loadConfigFile(*configPath)

	cfg := &Config{}

	// Auth method: flag/env > config file > default (azcli)
	authStr := resolve("AZURE_DEVOPS_AUTH_METHOD", "", fileVals)
	switch strings.ToLower(authStr) {
	case "pat":
		cfg.AuthMethod = AuthPAT
	case "azcli", "":
		cfg.AuthMethod = AuthAzCLI
	default:
		return nil, fmt.Errorf("unknown auth method %q: must be \"pat\" or \"azcli\"", authStr)
	}

	// Org URL — used to derive org name and base URL
	orgURL := resolve("AZURE_DEVOPS_ORG_URL", "", fileVals)

	// Org name — can come from URL, env, or flag
	orgName := resolve("AZDO_ORG", *org, fileVals)

	// If we have an org URL, parse org and project from it
	if orgURL != "" {
		parsedOrg, parsedProject := parseOrgURL(orgURL)
		if parsedOrg != "" && orgName == "" {
			orgName = parsedOrg
		}
		cfg.OrgURL = orgURL
		// If project was embedded in the URL (e.g. https://pdidev.visualstudio.com/PDI)
		if parsedProject != "" {
			cfg.Project = parsedProject
		}
	}

	cfg.Org = orgName

	// Project: flag/env > URL-parsed > config file
	projectVal := resolve("AZURE_DEVOPS_DEFAULT_PROJECT", *project, fileVals)
	if projectVal == "" {
		projectVal = resolve("AZDO_PROJECT", "", fileVals)
	}
	if projectVal != "" {
		cfg.Project = projectVal
	}

	// PAT
	if cfg.AuthMethod == AuthPAT {
		cfg.PAT = resolve("AZURE_DEVOPS_PAT", "", fileVals)
		if cfg.PAT == "" {
			return nil, fmt.Errorf("PAT auth requires AZURE_DEVOPS_PAT in config file or environment")
		}
	}

	// Validate
	if cfg.Org == "" {
		return nil, fmt.Errorf("organization is required: set AZURE_DEVOPS_ORG_URL or AZDO_ORG in config/env, or use --org flag")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required: set AZURE_DEVOPS_DEFAULT_PROJECT or AZDO_PROJECT in config/env, or use --project flag")
	}

	return cfg, nil
}

// resolve returns the first non-empty value from: flag, env var, config file.
func resolve(envKey, flagVal string, fileVals map[string]string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fileVals[envKey]
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
		// https://orgname.visualstudio.com/project
		org = strings.TrimSuffix(u.Host, ".visualstudio.com")
		if len(parts) > 0 && parts[0] != "" {
			project = parts[0]
		}
		return org, project
	}

	if u.Host == "dev.azure.com" {
		// https://dev.azure.com/orgname/project
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

// loadConfigFile reads a key=value config file.
// Looks in: provided path > ~/.config/azboard/config.env
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
		return vals // File doesn't exist, that's fine
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		// Strip surrounding quotes
		value = strings.Trim(value, `"'`)

		vals[key] = value
	}

	return vals
}

// ConfigFilePath returns the default config file path for display purposes.
func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.config/azboard/config.env"
	}
	return filepath.Join(home, ".config", "azboard", "config.env")
}

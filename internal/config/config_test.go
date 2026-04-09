package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValidAzCLI(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthMethod != AuthAzCLI {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, AuthAzCLI)
	}
	if cfg.Org != "myorg" {
		t.Errorf("Org = %q, want %q", cfg.Org, "myorg")
	}
	if cfg.Project != "MyProject" {
		t.Errorf("Project = %q, want %q", cfg.Project, "MyProject")
	}
}

func TestLoadValidPAT(t *testing.T) {
	path := writeTestConfig(t, `{
		"auth_method": "pat",
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject",
		"pat": "secret-token"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthMethod != AuthPAT {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, AuthPAT)
	}
	if cfg.PAT != "secret-token" {
		t.Errorf("PAT = %q, want %q", cfg.PAT, "secret-token")
	}
}

func TestLoadPATMissingToken(t *testing.T) {
	path := writeTestConfig(t, `{
		"auth_method": "pat",
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject"
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for PAT auth without token")
	}
}

func TestLoadMissingOrg(t *testing.T) {
	path := writeTestConfig(t, `{
		"project": "MyProject"
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing org")
	}
}

func TestLoadMissingProject(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg"
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestLoadInvalidAuthMethod(t *testing.T) {
	path := writeTestConfig(t, `{
		"auth_method": "oauth",
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject"
	}`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid auth method")
	}
}

func TestLoadDefaultValues(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultMergeStrategy != "squash" {
		t.Errorf("DefaultMergeStrategy = %q, want %q", cfg.DefaultMergeStrategy, "squash")
	}
	if len(cfg.WorkItemTypes) != 5 {
		t.Errorf("WorkItemTypes length = %d, want 5", len(cfg.WorkItemTypes))
	}
}

func TestLoadCustomValues(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg",
		"project": "MyProject",
		"repos": ["repo1", "repo2"],
		"work_item_types": ["Bug", "Task"],
		"default_merge_strategy": "rebase",
		"area_path": "Project\\Team"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Repos) != 2 || cfg.Repos[0] != "repo1" {
		t.Errorf("Repos = %v, want [repo1 repo2]", cfg.Repos)
	}
	if len(cfg.WorkItemTypes) != 2 {
		t.Errorf("WorkItemTypes = %v, want [Bug Task]", cfg.WorkItemTypes)
	}
	if cfg.DefaultMergeStrategy != "rebase" {
		t.Errorf("DefaultMergeStrategy = %q, want %q", cfg.DefaultMergeStrategy, "rebase")
	}
	if cfg.AreaPath != "Project\\Team" {
		t.Errorf("AreaPath = %q, want %q", cfg.AreaPath, "Project\\Team")
	}
}

func TestLoadProjectFromURL(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg/InlineProject"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project != "InlineProject" {
		t.Errorf("Project = %q, want %q", cfg.Project, "InlineProject")
	}
}

func TestLoadProjectOverridesURL(t *testing.T) {
	path := writeTestConfig(t, `{
		"org_url": "https://dev.azure.com/myorg/FromURL",
		"project": "Explicit"
	}`)
	cfg, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Project != "Explicit" {
		t.Errorf("Project = %q, want %q", cfg.Project, "Explicit")
	}
}

func TestParseOrgURLDevAzure(t *testing.T) {
	org, project := parseOrgURL("https://dev.azure.com/myorg")
	if org != "myorg" {
		t.Errorf("org = %q, want %q", org, "myorg")
	}
	if project != "" {
		t.Errorf("project = %q, want empty", project)
	}
}

func TestParseOrgURLDevAzureWithProject(t *testing.T) {
	org, project := parseOrgURL("https://dev.azure.com/myorg/myproject")
	if org != "myorg" {
		t.Errorf("org = %q, want %q", org, "myorg")
	}
	if project != "myproject" {
		t.Errorf("project = %q, want %q", project, "myproject")
	}
}

func TestParseOrgURLVisualStudio(t *testing.T) {
	org, project := parseOrgURL("https://myorg.visualstudio.com")
	if org != "myorg" {
		t.Errorf("org = %q, want %q", org, "myorg")
	}
	if project != "" {
		t.Errorf("project = %q, want empty", project)
	}
}

func TestParseOrgURLVisualStudioWithProject(t *testing.T) {
	org, project := parseOrgURL("https://myorg.visualstudio.com/myproject")
	if org != "myorg" {
		t.Errorf("org = %q, want %q", org, "myorg")
	}
	if project != "myproject" {
		t.Errorf("project = %q, want %q", project, "myproject")
	}
}

func TestParseOrgURLTrailingSlash(t *testing.T) {
	org, _ := parseOrgURL("https://dev.azure.com/myorg/")
	if org != "myorg" {
		t.Errorf("org = %q, want %q", org, "myorg")
	}
}

func TestParseOrgURLUnknownHost(t *testing.T) {
	org, project := parseOrgURL("https://example.com/foo")
	if org != "" || project != "" {
		t.Errorf("expected empty org/project for unknown host, got %q/%q", org, project)
	}
}

func TestConfigFilePath(t *testing.T) {
	path := ConfigFilePath()
	if !filepath.IsAbs(path) && path != "~/.config/azboard/config.json" {
		t.Errorf("ConfigFilePath() = %q, expected absolute path or fallback", path)
	}
	if filepath.Base(path) != "config.json" {
		t.Errorf("ConfigFilePath() base = %q, want config.json", filepath.Base(path))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := writeTestConfig(t, `not json`)
	_, err := LoadFromFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

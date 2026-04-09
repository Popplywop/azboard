package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/colorprofile"
	"github.com/popplywop/azboard/internal/api"
	"github.com/popplywop/azboard/internal/config"
	"github.com/popplywop/azboard/internal/ui"

	tea "charm.land/bubbletea/v2"
)

// version is set at build time via -ldflags "-X main.version=v0.0.1".
var version = "dev"

func main() {
	// Parse optional flags before loading config.
	var jumpToPRID int
	var demoMode bool
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "-version", "-v":
			fmt.Println("azboard", version)
			os.Exit(0)
		case "--pr":
			if i+1 < len(args) {
				id, err := strconv.Atoi(args[i+1])
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error: --pr requires an integer PR ID\n")
					os.Exit(1)
				}
				jumpToPRID = id
				i++
			}
		case "--demo":
			demoMode = true
		}
	}

	var client api.Clienter
	var org, project, orgURL string
	var repos, workItemTypes []string
	var defaultMergeStrategy, areaPath string

	if demoMode {
		client = api.NewMockClient()
		org = "contoso"
		project = "Contoso"
		orgURL = "https://dev.azure.com/contoso"
		repos = []string{"inventory-api", "web-portal", "auth-service"}
		workItemTypes = []string{"Bug", "User Story", "Task", "Feature", "Epic"}
		defaultMergeStrategy = "squash"
	} else {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
			fmt.Fprintf(os.Stderr, "Create a config file at %s:\n\n", config.ConfigFilePath())
			fmt.Fprintf(os.Stderr, "    {\n")
			fmt.Fprintf(os.Stderr, "      \"auth_method\": \"pat\",\n")
			fmt.Fprintf(os.Stderr, "      \"org_url\": \"https://dev.azure.com/yourorg\",\n")
			fmt.Fprintf(os.Stderr, "      \"project\": \"YourProject\",\n")
			fmt.Fprintf(os.Stderr, "      \"pat\": \"your-personal-access-token\"\n")
			fmt.Fprintf(os.Stderr, "    }\n")
			os.Exit(1)
		}

		realClient, err := api.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			if cfg.AuthMethod == config.AuthAzCLI {
				fmt.Fprintf(os.Stderr, "\nMake sure you are logged in with: az login\n")
			} else {
				fmt.Fprintf(os.Stderr, "\nCheck your PAT in %s\n", config.ConfigFilePath())
			}
			os.Exit(1)
		}
		client = realClient
		org = cfg.Org
		project = cfg.Project
		orgURL = cfg.OrgURL
		repos = cfg.Repos
		workItemTypes = cfg.WorkItemTypes
		defaultMergeStrategy = cfg.DefaultMergeStrategy
		areaPath = cfg.AreaPath
	}

	model := ui.NewAppModel(
		client,
		org,
		project,
		orgURL,
		repos,
		workItemTypes,
		defaultMergeStrategy,
		areaPath,
		jumpToPRID,
		version,
	)

	p := tea.NewProgram(model, tea.WithColorProfile(colorprofile.TrueColor))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %s\n", err)
		os.Exit(1)
	}
}

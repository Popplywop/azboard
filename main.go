package main

import (
	"fmt"
	"os"

	"azboard/internal/api"
	"azboard/internal/config"
	"azboard/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		fmt.Fprintf(os.Stderr, "Setup:\n")
		fmt.Fprintf(os.Stderr, "  Create a config file at %s:\n\n", config.ConfigFilePath())
		fmt.Fprintf(os.Stderr, "    # Auth: \"pat\" or \"azcli\" (default: azcli)\n")
		fmt.Fprintf(os.Stderr, "    AZURE_DEVOPS_AUTH_METHOD=pat\n")
		fmt.Fprintf(os.Stderr, "    AZURE_DEVOPS_ORG_URL=https://dev.azure.com/yourorg\n")
		fmt.Fprintf(os.Stderr, "    AZURE_DEVOPS_PAT=your-personal-access-token\n")
		fmt.Fprintf(os.Stderr, "    AZURE_DEVOPS_DEFAULT_PROJECT=YourProject\n\n")
		fmt.Fprintf(os.Stderr, "  Or use environment variables / flags:\n")
		fmt.Fprintf(os.Stderr, "    azboard --org <organization> --project <project>\n")
		os.Exit(1)
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		if cfg.AuthMethod == config.AuthAzCLI {
			fmt.Fprintf(os.Stderr, "\nMake sure you are logged in with: az login\n")
		} else {
			fmt.Fprintf(os.Stderr, "\nCheck your PAT in %s\n", config.ConfigFilePath())
		}
		os.Exit(1)
	}

	model := ui.NewAppModel(client, cfg.Org, cfg.Project)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %s\n", err)
		os.Exit(1)
	}
}

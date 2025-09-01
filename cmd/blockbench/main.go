package main

import (
	"fmt"
	"os"

	"github.com/makutaku/blockbench/internal/cli"
	"github.com/makutaku/blockbench/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "blockbench",
	Short: "A CLI tool for managing Minecraft Bedrock Edition addons",
	Long: `Blockbench is a command-line tool for managing Minecraft Bedrock Edition addons on servers.
It provides functionality to install, uninstall, and list addons with safety features like
automatic backups, rollback on failures, and dry-run mode for testing.`,
	Version: version.GetVersionString(),
}

func init() {
	rootCmd.PersistentFlags().Bool("dry-run", false, "Perform a dry run without making actual changes")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(cli.NewInstallCommand())
	rootCmd.AddCommand(cli.NewUninstallCommand())
	rootCmd.AddCommand(cli.NewListCommand())
	rootCmd.AddCommand(cli.NewVersionCommand())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

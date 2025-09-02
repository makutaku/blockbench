package cli

import (
	"fmt"
	"path/filepath"

	"github.com/makutaku/blockbench/internal/addon"
	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/spf13/cobra"
)

func NewInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [addon-file] [server-path]",
		Short: "Install a Minecraft Bedrock addon to a server",
		Long: `Install a Minecraft Bedrock addon to a server.

Supports both .mcaddon files (containing multiple packs) and individual .mcpack files.
The addon will be extracted, validated, and installed with automatic backup creation.`,
		Args: cobra.ExactArgs(2),
		RunE: runInstall,
	}

	cmd.Flags().Bool("force", false, "Force installation even if conflicts are detected")
	cmd.Flags().String("backup-dir", "", "Custom backup directory (default: server-path/backups)")
	cmd.Flags().Bool("interactive", false, "Interactive mode - confirm each step before proceeding")

	return cmd
}

func runInstall(cmd *cobra.Command, args []string) error {
	addonFile := args[0]
	serverPath := args[1]

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	force, _ := cmd.Flags().GetBool("force")
	interactive, _ := cmd.Flags().GetBool("interactive")
	backupDir, _ := cmd.Flags().GetString("backup-dir")

	// Set default backup directory
	if backupDir == "" {
		backupDir = filepath.Join(serverPath, "backups")
	}

	// Create server instance
	server, err := minecraft.NewServer(serverPath)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Create installer
	installer := addon.NewInstaller(server, backupDir)

	// Set up install options
	options := addon.InstallOptions{
		DryRun:      dryRun,
		Verbose:     verbose,
		BackupDir:   backupDir,
		ForceUpdate: force,
		Interactive: interactive,
	}

	// Perform installation
	result, err := installer.InstallAddon(addonFile, options)

	// Display results
	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}

	if result.Success {
		if dryRun {
			fmt.Println("DRY RUN: Installation would succeed")
		} else {
			fmt.Printf("Successfully installed addon with %d pack(s)\n", len(result.InstalledPacks))
			if verbose {
				for _, pack := range result.InstalledPacks {
					fmt.Printf("  - %s\n", pack)
				}
			}
		}
		return nil
	}

	return err
}

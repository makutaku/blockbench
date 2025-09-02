package cli

import (
	"fmt"
	"path/filepath"

	"github.com/makutaku/blockbench/internal/addon"
	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/spf13/cobra"
)

func NewUninstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall [addon-name] [server-path]",
		Short: "Uninstall a Minecraft Bedrock addon from a server",
		Long: `Uninstall an addon from a Minecraft Bedrock server by name.
The addon will be safely removed with dependency checking and backup creation.`,
		Args: cobra.ExactArgs(2),
		RunE: runUninstall,
	}

	cmd.Flags().String("uuid", "", "Uninstall addon by UUID instead of name")
	cmd.Flags().String("backup-dir", "", "Custom backup directory (default: server-path/backups)")
	cmd.Flags().Bool("interactive", false, "Interactive mode - confirm each step before proceeding")

	return cmd
}

func runUninstall(cmd *cobra.Command, args []string) error {
	identifier := args[0]
	serverPath := args[1]

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	interactive, _ := cmd.Flags().GetBool("interactive")
	uuid, _ := cmd.Flags().GetString("uuid")
	backupDir, _ := cmd.Flags().GetString("backup-dir")

	// Set default backup directory
	if backupDir == "" {
		backupDir = filepath.Join(serverPath, "backups")
	}

	// Determine if we're searching by UUID
	byUUID := uuid != ""
	if byUUID {
		identifier = uuid
	}

	// Create server instance
	server, err := minecraft.NewServer(serverPath)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Create uninstaller
	uninstaller := addon.NewUninstaller(server, backupDir)

	// Set up uninstall options
	options := addon.UninstallOptions{
		DryRun:      dryRun,
		Verbose:     verbose,
		BackupDir:   backupDir,
		ByUUID:      byUUID,
		Interactive: interactive,
	}

	// Perform uninstallation
	result, err := uninstaller.UninstallAddon(identifier, options)

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
			fmt.Println("DRY RUN: Uninstallation would succeed")
		} else {
			fmt.Printf("Successfully uninstalled %d pack(s)\n", len(result.RemovedPacks))
			if verbose {
				for _, pack := range result.RemovedPacks {
					fmt.Printf("  - %s\n", pack)
				}
			}
		}
		return nil
	}

	return err
}

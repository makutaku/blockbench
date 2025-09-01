package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/makutaku/blockbench/internal/minecraft"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [server-path]",
		Short: "List installed Minecraft Bedrock addons",
		Long: `List all addons currently installed on a Minecraft Bedrock server.
Shows addon names, UUIDs, versions, and types (behavior/resource packs).`,
		Args: cobra.ExactArgs(1),
		RunE: runList,
	}

	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	serverPath := args[0]
	
	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if verbose {
		fmt.Printf("Listing addons for server at %s\n", serverPath)
	}

	// Create server instance
	server, err := minecraft.NewServer(serverPath)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Get installed packs
	installedPacks, err := server.ListInstalledPacks()
	if err != nil {
		return fmt.Errorf("failed to list installed packs: %w", err)
	}

	if len(installedPacks) == 0 {
		if !jsonOutput {
			fmt.Println("No addons installed")
		} else {
			fmt.Println("[]")
		}
		return nil
	}

	if jsonOutput {
		// Output as JSON
		data, err := json.MarshalIndent(installedPacks, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Output as table
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tUUID\tVERSION\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t----\t-------\t-----------")

	for _, pack := range installedPacks {
		name := pack.Name
		if name == "" {
			name = fmt.Sprintf("Pack-%s", pack.PackID[:8])
		}

		description := pack.Description
		if len(description) > 50 {
			description = description[:47] + "..."
		}

		version := fmt.Sprintf("%d.%d.%d", pack.Version[0], pack.Version[1], pack.Version[2])
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", 
			name, pack.Type, pack.PackID, version, description)
	}

	w.Flush()

	if verbose {
		fmt.Printf("\nTotal: %d pack(s) installed\n", len(installedPacks))
	}

	return nil
}
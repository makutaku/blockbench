package cli

import (
	"encoding/json"
	"fmt"

	"github.com/makutaku/blockbench/internal/version"
	"github.com/spf13/cobra"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information including build details",
		RunE:  runVersion,
	}

	cmd.Flags().Bool("json", false, "Output version information in JSON format")
	cmd.Flags().Bool("short", false, "Output only the version number")

	return cmd
}

func runVersion(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")
	shortOutput, _ := cmd.Flags().GetBool("short")

	versionInfo := version.GetVersion()

	if shortOutput {
		fmt.Println(versionInfo.Version)
		return nil
	}

	if jsonOutput {
		data, err := json.MarshalIndent(versionInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Println(version.GetFullVersionString())
	return nil
}

package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/makutaku/blockbench/internal/addon"
	"github.com/makutaku/blockbench/internal/minecraft"
	"github.com/spf13/cobra"
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
	cmd.Flags().Bool("grouped", false, "Group packs by dependency relationships")
	cmd.Flags().Bool("tree", false, "Show dependency tree visualization")
	cmd.Flags().Bool("standalone", false, "Show only standalone packs (no dependencies)")
	cmd.Flags().Bool("roots", false, "Show only root packs (packs that others depend on)")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	serverPath := args[0]

	verbose, _ := cmd.Flags().GetBool("verbose")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	grouped, _ := cmd.Flags().GetBool("grouped")
	tree, _ := cmd.Flags().GetBool("tree")
	standaloneOnly, _ := cmd.Flags().GetBool("standalone")
	rootsOnly, _ := cmd.Flags().GetBool("roots")

	if verbose {
		fmt.Printf("Listing addons for server at %s\n", serverPath)
	}

	// Create server instance
	server, err := minecraft.NewServer(serverPath)
	if err != nil {
		return fmt.Errorf("failed to initialize server: %w", err)
	}

	// Check if dependency analysis is needed
	if grouped || tree || standaloneOnly || rootsOnly {
		return runListWithDependencies(server, jsonOutput, verbose, grouped, tree, standaloneOnly, rootsOnly)
	}

	// Default behavior - simple flat list
	return runSimpleList(server, jsonOutput, verbose)
}

func runSimpleList(server *minecraft.Server, jsonOutput, verbose bool) error {
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
	renderSimpleTable(installedPacks)

	if verbose {
		fmt.Printf("\nTotal: %d pack(s) installed\n", len(installedPacks))
	}

	return nil
}

func runListWithDependencies(server *minecraft.Server, jsonOutput, verbose, grouped, tree, standaloneOnly, rootsOnly bool) error {
	// Create dependency analyzer
	analyzer := addon.NewDependencyAnalyzer(server)

	// Analyze dependencies
	dependencyGroup, err := analyzer.AnalyzeDependencies()
	if err != nil {
		return fmt.Errorf("failed to analyze dependencies: %w", err)
	}

	// Handle JSON output for dependency data
	if jsonOutput {
		return outputDependencyJSON(dependencyGroup, standaloneOnly, rootsOnly)
	}

	// Handle different display modes
	if tree {
		return renderTreeView(analyzer, dependencyGroup)
	}

	if grouped {
		return renderGroupedView(dependencyGroup, standaloneOnly, rootsOnly, verbose)
	}

	if standaloneOnly {
		return renderStandaloneView(dependencyGroup, verbose)
	}

	if rootsOnly {
		return renderRootsView(dependencyGroup, verbose)
	}

	// Default to grouped view if dependency analysis was requested
	return renderGroupedView(dependencyGroup, false, false, verbose)
}

func renderSimpleTable(packs []minecraft.InstalledPack) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tUUID\tVERSION\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t----\t-------\t-----------")

	for _, pack := range packs {
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
}

func renderGroupedView(group *addon.DependencyGroup, standaloneOnly, rootsOnly bool, verbose bool) error {
	totalPacks := len(group.RootPacks) + len(group.DependentPacks) + len(group.StandalonePacks)

	if !standaloneOnly && len(group.RootPacks) > 0 {
		fmt.Printf("🎯 ROOT PACKS (%d)\n", len(group.RootPacks))
		fmt.Println("Packs that other packs depend on:")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tDEPENDENTS\tMODULES")
		fmt.Fprintln(w, "----\t----\t-------\t----------\t-------")

		for _, rel := range group.RootPacks {
			name := rel.Pack.Name
			if name == "" {
				name = fmt.Sprintf("Pack-%s", rel.Pack.PackID[:8])
			}
			version := fmt.Sprintf("%d.%d.%d", rel.Pack.Version[0], rel.Pack.Version[1], rel.Pack.Version[2])
			dependentCount := fmt.Sprintf("%d pack(s)", len(rel.Dependents))
			modules := strings.Join(rel.Modules, ", ")
			if len(modules) > 30 {
				modules = modules[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name, rel.Pack.Type, version, dependentCount, modules)
		}
		w.Flush()
		fmt.Println()
	}

	if !standaloneOnly && !rootsOnly && len(group.DependentPacks) > 0 {
		fmt.Printf("📦 DEPENDENT PACKS (%d)\n", len(group.DependentPacks))
		fmt.Println("Packs that depend on other packs:")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tDEPENDS ON\tMODULES")
		fmt.Fprintln(w, "----\t----\t-------\t----------\t-------")

		for _, rel := range group.DependentPacks {
			name := rel.Pack.Name
			if name == "" {
				name = fmt.Sprintf("Pack-%s", rel.Pack.PackID[:8])
			}
			version := fmt.Sprintf("%d.%d.%d", rel.Pack.Version[0], rel.Pack.Version[1], rel.Pack.Version[2])
			dependencyCount := fmt.Sprintf("%d pack(s)", len(rel.Dependencies))
			modules := strings.Join(rel.Modules, ", ")
			if len(modules) > 30 {
				modules = modules[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				name, rel.Pack.Type, version, dependencyCount, modules)
		}
		w.Flush()
		fmt.Println()
	}

	if !rootsOnly && len(group.StandalonePacks) > 0 {
		fmt.Printf("🎯 STANDALONE PACKS (%d)\n", len(group.StandalonePacks))
		fmt.Println("Packs with no dependencies or dependents:")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tMODULES")
		fmt.Fprintln(w, "----\t----\t-------\t-------")

		for _, rel := range group.StandalonePacks {
			name := rel.Pack.Name
			if name == "" {
				name = fmt.Sprintf("Pack-%s", rel.Pack.PackID[:8])
			}
			version := fmt.Sprintf("%d.%d.%d", rel.Pack.Version[0], rel.Pack.Version[1], rel.Pack.Version[2])
			modules := strings.Join(rel.Modules, ", ")
			if len(modules) > 30 {
				modules = modules[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				name, rel.Pack.Type, version, modules)
		}
		w.Flush()
		fmt.Println()
	}

	if verbose {
		fmt.Printf("Total: %d pack(s) installed\n", totalPacks)
	}

	return nil
}

func renderStandaloneView(group *addon.DependencyGroup, verbose bool) error {
	if len(group.StandalonePacks) == 0 {
		fmt.Println("No standalone packs found")
		return nil
	}

	fmt.Printf("🎯 STANDALONE PACKS (%d)\n", len(group.StandalonePacks))
	renderSimpleRelationshipTable(group.StandalonePacks)

	if verbose {
		fmt.Printf("\nShowing %d standalone pack(s)\n", len(group.StandalonePacks))
	}

	return nil
}

func renderRootsView(group *addon.DependencyGroup, verbose bool) error {
	if len(group.RootPacks) == 0 {
		fmt.Println("No root packs found")
		return nil
	}

	fmt.Printf("🎯 ROOT PACKS (%d)\n", len(group.RootPacks))
	fmt.Println("Packs that other packs depend on:")
	renderSimpleRelationshipTable(group.RootPacks)

	if verbose {
		fmt.Printf("\nShowing %d root pack(s)\n", len(group.RootPacks))
	}

	return nil
}

func renderSimpleRelationshipTable(relationships []addon.PackRelationship) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tUUID\tVERSION\tMODULES")
	fmt.Fprintln(w, "----\t----\t----\t-------\t-------")

	for _, rel := range relationships {
		name := rel.Pack.Name
		if name == "" {
			name = fmt.Sprintf("Pack-%s", rel.Pack.PackID[:8])
		}
		version := fmt.Sprintf("%d.%d.%d", rel.Pack.Version[0], rel.Pack.Version[1], rel.Pack.Version[2])
		modules := strings.Join(rel.Modules, ", ")
		if len(modules) > 40 {
			modules = modules[:37] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			name, rel.Pack.Type, rel.Pack.PackID, version, modules)
	}
	w.Flush()
}

func outputDependencyJSON(group *addon.DependencyGroup, standaloneOnly, rootsOnly bool) error {
	type JSONOutput struct {
		RootPacks       []addon.PackRelationship `json:"root_packs,omitempty"`
		DependentPacks  []addon.PackRelationship `json:"dependent_packs,omitempty"`
		StandalonePacks []addon.PackRelationship `json:"standalone_packs,omitempty"`
	}

	output := JSONOutput{}

	if !standaloneOnly {
		output.RootPacks = group.RootPacks
	}
	if !standaloneOnly && !rootsOnly {
		output.DependentPacks = group.DependentPacks
	}
	if !rootsOnly {
		output.StandalonePacks = group.StandalonePacks
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func renderTreeView(analyzer *addon.DependencyAnalyzer, group *addon.DependencyGroup) error {
	fmt.Println("📦 ADDON DEPENDENCY TREE")
	fmt.Println()

	tree := analyzer.GetDependencyTree(group)

	// Sort root packs for consistent output
	var roots []addon.PackRelationship
	roots = append(roots, group.RootPacks...)
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Pack.Name < roots[j].Pack.Name
	})

	for i, root := range roots {
		isLast := i == len(roots)-1
		renderTreeNode(root, tree[root.Pack.PackID], "", isLast)
	}

	// Show standalone packs as separate trees
	if len(group.StandalonePacks) > 0 {
		fmt.Println("🎯 STANDALONE:")
		for i, standalone := range group.StandalonePacks {
			isLast := i == len(group.StandalonePacks)-1
			renderTreeNode(standalone, []addon.PackRelationship{}, "", isLast)
		}
	}

	return nil
}

func renderTreeNode(pack addon.PackRelationship, children []addon.PackRelationship, prefix string, isLast bool) {
	// Determine the tree symbols
	var nodeSymbol, childPrefix string
	if isLast {
		nodeSymbol = "└── "
		childPrefix = prefix + "    "
	} else {
		nodeSymbol = "├── "
		childPrefix = prefix + "│   "
	}

	// Pack type emoji
	emoji := "📦"
	if pack.Pack.Type == minecraft.PackTypeResource {
		emoji = "🎨"
	}

	// Pack name and info
	name := pack.Pack.Name
	if name == "" {
		name = fmt.Sprintf("Pack-%s", pack.Pack.PackID[:8])
	}
	version := fmt.Sprintf("v%d.%d.%d", pack.Pack.Version[0], pack.Pack.Version[1], pack.Pack.Version[2])

	// Show modules if any
	moduleInfo := ""
	if len(pack.Modules) > 0 {
		moduleInfo = fmt.Sprintf(" [modules: %s]", strings.Join(pack.Modules, ", "))
	}

	fmt.Printf("%s%s%s %s (%s)%s\n", prefix, nodeSymbol, emoji, name, version, moduleInfo)

	// Render children
	for i, child := range children {
		isLastChild := i == len(children)-1
		renderTreeNode(child, []addon.PackRelationship{}, childPrefix, isLastChild)
	}
}

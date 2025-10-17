package main

import (
	"fmt"
	"log"
	"strings"

	"vuDataSim/src/logger"
	"vuDataSim/src/node_control"
)

// handleNodeManagementCLI handles CLI commands for node management
func handleNodeManagementCLI(command string, args []string) bool {
	switch command {
	case "add":
		handleAddNodeCLI(args)
		return true
	case "remove":
		handleRemoveNodeCLI(args)
		return true
	case "enable":
		handleEnableNodeCLI(args)
		return true
	case "disable":
		handleDisableNodeCLI(args)
		return true
	case "list":
		handleListNodesCLI()
		return true
	case "list-enabled":
		handleListEnabledNodesCLI()
		return true
	case "web":
		// Continue to web server mode
		return false
	default:
		return false // Not a node management command
	}
}

func handleAddNodeCLI(args []string) {
	if len(args) < 6 {
		log.Fatal("Usage: vuDataSim-manager add <name> <host> <user> <key_path> <conf_dir> <binary_dir> [description] [enabled]")
	}

	name := args[0]
	host := args[1]
	user := args[2]
	keyPath := args[3]
	confDir := args[4]
	binaryDir := args[5]

	description := ""
	if len(args) > 6 {
		description = strings.Join(args[6:len(args)-1], " ")
	}

	enabled := true
	if len(args) > 7 {
		enabled = args[len(args)-1] == "true"
	}

	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
		log.Println("Continuing with default configuration")
	}

	err = nodeManager.AddNode(node_control.AddNodeRequest{
		Name:        name,
		Host:        host,
		User:        user,
		KeyPath:     keyPath,
		ConfDir:     confDir,
		BinaryDir:   binaryDir,
		Description: description,
		Enabled:     enabled,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func handleRemoveNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager remove <name>")
	}

	name := args[0]

	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Str("module", "cli_handlers").Err(err).Msg("Failed to load nodes config")
		logger.Info().Str("module", "cli_handlers").Msg("Continuing with default configuration")
	}

	err = nodeManager.RemoveNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleEnableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager enable <name>")
	}

	name := args[0]

	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Str("module", "cli_handlers").Err(err).Msg("Failed to load nodes config")
		logger.Info().Str("module", "cli_handlers").Msg("Continuing with default configuration")
	}

	err = nodeManager.EnableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleDisableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager disable <name>")
	}

	name := args[0]

	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		logger.Warn().Str("module", "cli_handlers").Err(err).Msg("Failed to load nodes config")
		logger.Info().Str("module", "cli_handlers").Msg("Continuing with default configuration")
	}

	err = nodeManager.DisableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleListNodesCLI() {
	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
		log.Println("Continuing with default configuration")
	}

	nodes := nodeManager.GetNodes()
	if len(nodes) == 0 {
		fmt.Println("No nodes configured")
		return
	}

	fmt.Println("Configured Nodes:")
	fmt.Println("================")

	for name, config := range nodes {
		status := "Disabled"
		if config.Enabled {
			status = "Enabled"
		}

		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
	}
}

func handleListEnabledNodesCLI() {
	// Create node manager instance and load configuration
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config: %v", err)
		log.Println("Continuing with default configuration")
	}

	enabledNodes := nodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		fmt.Println("No enabled nodes")
		return
	}

	fmt.Println("Enabled Nodes:")
	fmt.Println("==============")

	for name, config := range enabledNodes {
		fmt.Printf("Node: %s\n", name)
		fmt.Printf("  Host: %s\n", config.Host)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Description: %s\n", config.Description)
		fmt.Printf("  Binary Dir: %s\n", config.BinaryDir)
		fmt.Printf("  Conf Dir: %s\n", config.ConfDir)
		fmt.Println()
	}
}

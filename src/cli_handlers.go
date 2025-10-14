package main

import (
	"fmt"
	"log"
	"strings"
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

	err := nodeManager.AddNode(name, host, user, keyPath, confDir, binaryDir, description, enabled)
	if err != nil {
		log.Fatal(err)
	}
}

func handleRemoveNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager remove <name>")
	}

	name := args[0]
	err := nodeManager.RemoveNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleEnableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager enable <name>")
	}

	name := args[0]
	err := nodeManager.EnableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleDisableNodeCLI(args []string) {
	if len(args) != 1 {
		log.Fatal("Usage: vuDataSim-manager disable <name>")
	}

	name := args[0]
	err := nodeManager.DisableNode(name)
	if err != nil {
		log.Fatal(err)
	}
}

func handleListNodesCLI() {
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

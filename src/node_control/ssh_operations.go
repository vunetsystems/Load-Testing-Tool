package node_control

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	SSHOptionStrictHostKeyChecking = "StrictHostKeyChecking=no"
	SSHOptionUserKnownHostsFile    = "UserKnownHostsFile=/dev/null"
	SSHOptionConnectTimeout        = "ConnectTimeout=10"
	SSHOptionLogLevel              = "LogLevel=ERROR"
)

func (nm *NodeManager) SSHExecWithOutput(nodeConfig NodeConfig, command string) (string, error) {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", SSHOptionStrictHostKeyChecking,
		"-o", SSHOptionUserKnownHostsFile,
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("SSH command failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (nm *NodeManager) copyFilesToNode(nodeName string, nodeConfig NodeConfig) error {
	localMainBinary := "src/finalvudatasim"
	localMetricsBinary := "src/node_metrics_api/build/node_metrics_api"
	localConfDir := "src/conf.d"

	log.Printf("DEBUG: Deployment paths for node %s:", nodeName)
	log.Printf("  Main binary path: %s", localMainBinary)
	log.Printf("  Metrics binary path: %s", localMetricsBinary)
	log.Printf("  Conf dir path: %s", localConfDir)

	// Check if local files exist
	if _, err := os.Stat(localMainBinary); os.IsNotExist(err) {
		return fmt.Errorf("local main binary file %s not found", localMainBinary)
	}

	if _, err := os.Stat(localMetricsBinary); os.IsNotExist(err) {
		return fmt.Errorf("local metrics binary file %s not found", localMetricsBinary)
	}

	if _, err := os.Stat(localConfDir); os.IsNotExist(err) {
		return fmt.Errorf("local conf.d directory %s not found", localConfDir)
	}

	// Create remote directories
	err := nm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s %s", nodeConfig.BinaryDir, nodeConfig.ConfDir))
	if err != nil {
		return fmt.Errorf("failed to create remote directories: %v", err)
	}

	// Copy main binary file
	log.Printf("Copying main binary from %s to %s", localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	err = nm.scpCopy(nodeConfig, localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim"))
	if err != nil {
		log.Printf("ERROR: Failed to copy main binary: %v", err)
		return fmt.Errorf("failed to copy main binary: %v", err)
	}
	log.Printf("✓ Main binary copied successfully")

	// Copy metrics API binary
	log.Printf("Copying metrics binary from %s to %s", localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api"))
	err = nm.scpCopy(nodeConfig, localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api"))
	if err != nil {
		log.Printf("ERROR: Failed to copy metrics binary: %v", err)
		return fmt.Errorf("failed to copy metrics binary: %v", err)
	}
	log.Printf("✓ Metrics binary copied successfully")

	// Copy conf.d directory recursively
	log.Printf("Copying conf.d directory from %s to %s", localConfDir, nodeConfig.ConfDir)
	err = nm.scpCopyDir(nodeConfig, localConfDir, nodeConfig.ConfDir)
	if err != nil {
		log.Printf("ERROR: Failed to copy conf.d directory: %v", err)
		return fmt.Errorf("failed to copy conf.d directory: %v", err)
	}
	log.Printf("✓ Conf.d directory copied successfully")

	log.Printf("Successfully copied files to node %s", nodeName)
	return nil
}

func (nm *NodeManager) cleanupNodeFiles(nodeName string) error {
	nodeSnapshotDir := filepath.Join(nm.snapshotsDir, nodeName)
	nodeBackupDir := filepath.Join(nm.backupsDir, nodeName)

	if _, err := os.Stat(nodeSnapshotDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeSnapshotDir)
		if err != nil {
			return fmt.Errorf("failed to remove snapshot directory: %v", err)
		}
	}

	if _, err := os.Stat(nodeBackupDir); !os.IsNotExist(err) {
		err := os.RemoveAll(nodeBackupDir)
		if err != nil {
			return fmt.Errorf("failed to remove backup directory: %v", err)
		}
	}

	return nil
}

func (nm *NodeManager) scpCopyDir(nodeConfig NodeConfig, localDir, remoteDir string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", SSHOptionStrictHostKeyChecking,
		"-o", SSHOptionUserKnownHostsFile,
		"-r",
		localDir,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remoteDir),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP directory copy failed: %v", err)
	}

	return nil
}

func (nm *NodeManager) scpCopy(nodeConfig NodeConfig, localPath, remotePath string) error {
	log.Printf("DEBUG: SCP copying %s to %s@%s:%s", localPath, nodeConfig.User, nodeConfig.Host, remotePath)

	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", SSHOptionStrictHostKeyChecking,
		"-o", SSHOptionUserKnownHostsFile,
		"-o", SSHOptionConnectTimeout,
		"-o", SSHOptionLogLevel,
	}

	// Add -r only if localPath is a directory
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local path %s: %v", localPath, err)
	}
	if info.IsDir() {
		args = append(args, "-r")
		log.Printf("DEBUG: Copying directory with -r flag")
	}

	args = append(args, localPath, fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath))

	log.Printf("DEBUG: Executing SCP command: scp %v", args)

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Printf("ERROR: SCP command failed for %s: %v", localPath, err)
		return fmt.Errorf("SCP copy failed: %v", err)
	}

	log.Printf("DEBUG: SCP copy successful for %s", localPath)
	return nil
}

func (nm *NodeManager) sshExec(nodeConfig NodeConfig, command string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH command failed: %v", err)
	}

	return nil
}

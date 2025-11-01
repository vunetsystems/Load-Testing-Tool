package node_control

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"vuDataSim/src/logger"
)

const (
	SSHOptionStrictHostKeyChecking = "StrictHostKeyChecking=no"
	SSHOptionUserKnownHostsFile    = "UserKnownHostsFile=/dev/null"
	SSHOptionConnectTimeout        = "ConnectTimeout=10"
	SSHOptionLogLevel              = "LogLevel=ERROR"
)

func (nm *NodeManager) SSHExecWithOutput(nodeConfig NodeConfig, command string) (string, error) {
	// Acquire semaphore to limit concurrent SSH operations
	nm.sshSemaphore <- struct{}{}
	defer func() { <-nm.sshSemaphore }()
	
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", SSHOptionStrictHostKeyChecking,
		"-o", SSHOptionUserKnownHostsFile,
		"-o", SSHOptionConnectTimeout,
		"-o", SSHOptionLogLevel,
		nodeConfig.Host,
		command,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("SSH command timed out after 30 seconds: %s", command)
		}
		return "", fmt.Errorf("SSH command failed: %v", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (nm *NodeManager) copyFilesToNode(nodeName string, nodeConfig NodeConfig) error {
	logger.Info().Str("node", nodeName).Msg("📦 Starting file copy to node")

	localMainBinary := "src/migrate/finalvudatasim"
	localMetricsBinary := "src/node_metrics_api/build/node_metrics_api"
	localConfDir := "src/migrate/conf.d"

	checkPaths := map[string]string{
		"main_binary":    localMainBinary,
		"metrics_binary": localMetricsBinary,
		"conf_dir":       localConfDir,
	}

	for label, path := range checkPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			logger.Error().Err(err).Str("node", nodeName).Str("file", path).Msgf("Missing required %s", label)
			return fmt.Errorf("%s file %s not found", label, path)
		}
	}

	if err := nm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s %s", nodeConfig.BinaryDir, nodeConfig.ConfDir)); err != nil {
		return fmt.Errorf("failed to create remote directories: %v", err)
	}

	// Copy files
	filesToCopy := []struct {
		src, dst string
	}{
		{localMainBinary, filepath.Join(nodeConfig.BinaryDir, "finalvudatasim")},
		{localMetricsBinary, filepath.Join(nodeConfig.BinaryDir, "node_metrics_api")},
	}

	for _, f := range filesToCopy {
		if err := nm.scpCopy(nodeConfig, f.src, f.dst); err != nil {
			logger.Error().Err(err).Str("node", nodeName).Str("src", f.src).Msg("File copy failed")
			return fmt.Errorf("failed to copy %s: %v", f.src, err)
		}
		logger.Info().Str("node", nodeName).Str("dst", f.dst).Msg("✅ File copied successfully")
	}

	if err := nm.scpCopyDir(nodeConfig, localConfDir, nodeConfig.ConfDir); err != nil {
		logger.Error().Err(err).Str("node", nodeName).Msg("Failed to copy conf.d directory")
		return fmt.Errorf("failed to copy conf.d directory: %v", err)
	}
	logger.Info().Str("node", nodeName).Msg("✅ conf.d directory copied")

	// Make binaries executable
	chmodCmd := fmt.Sprintf("chmod +x %s/finalvudatasim %s/node_metrics_api", nodeConfig.BinaryDir, nodeConfig.BinaryDir)
	if err := nm.sshExec(nodeConfig, chmodCmd); err != nil {
		return fmt.Errorf("failed to chmod binaries: %v", err)
	}
	logger.Info().Str("node", nodeName).Msg("✅ Binaries made executable")

	// Start binaries in background
	startCmd := fmt.Sprintf("nohup %s/finalvudatasim > /tmp/finalvudatasim.log 2>&1 & nohup %s/node_metrics_api > /tmp/node_metrics_api.log 2>&1 &",
		nodeConfig.BinaryDir, nodeConfig.BinaryDir)

	if err := nm.sshExec(nodeConfig, startCmd); err != nil {
		logger.Error().Err(err).Str("node", nodeName).Msg("❌ Failed to start binaries")
		return fmt.Errorf("failed to start binaries: %v", err)
	}

	logger.Info().Str("node", nodeName).Msg("🚀 Node binaries started successfully")
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
	logger.Info().Str("localDir", localDir).Str("remoteDir", remoteDir).Str("host", nodeConfig.Host).Msg("SCP copying directory")

	// Acquire semaphore to limit concurrent SSH operations
	nm.sshSemaphore <- struct{}{}
	defer func() { <-nm.sshSemaphore }()

	// Expand ~ in key path
	expandedKeyPath := nodeConfig.KeyPath
	if strings.HasPrefix(expandedKeyPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		expandedKeyPath = strings.Replace(expandedKeyPath, "~", homeDir, 1)
		logger.Info().Str("originalKeyPath", nodeConfig.KeyPath).Str("expandedKeyPath", expandedKeyPath).Msg("SSH key path expansion for directory copy")
	}

	args := []string{
		"-i", expandedKeyPath,
		"-o", SSHOptionStrictHostKeyChecking,
		"-o", SSHOptionUserKnownHostsFile,
		"-o", SSHOptionConnectTimeout,
		"-o", SSHOptionLogLevel,
		"-r",
		localDir,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remoteDir),
	}

	logger.Info().Strs("args", args).Msg("Executing SCP directory command")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "scp", args...)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().Err(err).Str("localDir", localDir).Str("output", string(output)).Msg("SCP directory copy failed")
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("SCP directory copy timed out after 60 seconds")
		}
		return fmt.Errorf("SCP directory copy failed: %v, output: %s", err, string(output))
	}

	logger.Info().Str("localDir", localDir).Msg("SCP directory copy successful")
	return nil
}

func (nm *NodeManager) scpCopy(nodeConfig NodeConfig, localPath, remotePath string) error {
	logger.Info().Str("local", localPath).Str("remote", remotePath).Str("host", nodeConfig.Host).Str("keyPath", nodeConfig.KeyPath).Msg("SCP copying file")

	// Acquire semaphore to limit concurrent SSH operations
	nm.sshSemaphore <- struct{}{}
	defer func() { <-nm.sshSemaphore }()

	// Expand ~ in key path
	expandedKeyPath := nodeConfig.KeyPath
	if strings.HasPrefix(expandedKeyPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		expandedKeyPath = strings.Replace(expandedKeyPath, "~", homeDir, 1)
		logger.Info().Str("originalKeyPath", nodeConfig.KeyPath).Str("expandedKeyPath", expandedKeyPath).Msg("SSH key path expansion")
	}

	args := []string{
		"-i", expandedKeyPath,
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
		logger.Info().Str("path", localPath).Msg("Copying directory with -r flag")
	}

	// Construct remote destination with user@host format
	remoteDest := fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath)
	args = append(args, localPath, remoteDest)

	logger.Info().Strs("args", args).Msg("Executing SCP command")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "scp", args...)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().Err(err).Str("localPath", localPath).Str("output", string(output)).Msg("SCP command failed")
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("SCP copy timed out after 60 seconds for %s", localPath)
		}
		return fmt.Errorf("SCP copy failed: %v, output: %s", err, string(output))
	}

	logger.Info().Str("localPath", localPath).Msg("SCP copy successful")
	return nil
}

func (nm *NodeManager) sshExec(nodeConfig NodeConfig, command string) error {
	logger.Info().Str("host", nodeConfig.Host).Str("command", command).Msg("Executing SSH command")

	// Acquire semaphore to limit concurrent SSH operations
	nm.sshSemaphore <- struct{}{}
	defer func() { <-nm.sshSemaphore }()

	// Expand ~ in key path
	expandedKeyPath := nodeConfig.KeyPath
	if strings.HasPrefix(expandedKeyPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		expandedKeyPath = strings.Replace(expandedKeyPath, "~", homeDir, 1)
		logger.Info().Str("originalKeyPath", nodeConfig.KeyPath).Str("expandedKeyPath", expandedKeyPath).Msg("SSH key path expansion for SSH exec")
	}

	args := []string{
		"-i", expandedKeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	logger.Info().Strs("args", args).Msg("SSH command arguments")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", args...)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error().Err(err).Str("command", command).Str("output", string(output)).Msg("SSH command failed")
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("SSH command timed out after 30 seconds: %s", command)
		}
		return fmt.Errorf("SSH command failed: %v, output: %s", err, string(output))
	}

	logger.Info().Str("command", command).Msg("SSH command successful")
	return nil
}

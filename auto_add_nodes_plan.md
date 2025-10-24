# Auto-Add All Cluster Nodes Feature Plan

## Overview
Add functionality to automatically discover and add all nodes in a Kubernetes cluster using `kubectl get nodes -o wide` command. The tool runs on the master node and will SSH to each discovered node using default SSH configuration.

## Requirements
- Keep existing manual node addition functionality unchanged
- Add default SSH configuration fields to config.yaml
- Create new API endpoint for auto-discovering nodes
- Add frontend button to trigger auto-add functionality
- Parse kubectl output to extract node IPs
- Use existing node addition logic with default configs

## Implementation Details

### 1. Configuration Changes
Add default SSH settings to `src/configs/config.yaml`:
```yaml
ssh_defaults:
  user: vunet
  key_path: ~/.ssh/id_rsa
  conf_dir: /home
  binary_dir: /home/bin
```

### 2. Backend Changes

#### New API Endpoint
Add `HandleAPIAutoAddNodes` function to `src/handlers/nodes.go`:
- Execute `kubectl get nodes -o wide` command
- Parse output to extract INTERNAL-IP addresses
- Skip the master node (current host)
- For each discovered node, call existing `AddNode` functionality with default SSH config

#### Node Manager Updates
Update `src/node_control/node_manager.go`:
- Add method to get default SSH configuration from app config
- Ensure auto-added nodes use default paths

### 3. Frontend Changes

#### UI Updates
Add "Auto Add All Nodes" button to node management modal in `static/index.html`:
- Place near existing "Add Node" button
- Add appropriate icon and styling

#### JavaScript Updates
Update `static/node-management.js`:
- Add `autoAddAllNodes()` method
- Handle API response and refresh node table
- Show appropriate success/error notifications

### 4. Command Execution
- Use Go's `exec.Command` to run `kubectl get nodes -o wide`
- Parse tabular output to extract INTERNAL-IP column
- Filter out current master node IP

### 5. Integration
- Auto-added nodes will use default SSH user, key path, conf dir, and binary dir
- Node names will be generated from IPs or kubectl node names
- All nodes enabled by default

## Testing
- Test kubectl command execution and parsing
- Verify SSH connectivity to discovered nodes
- Ensure existing manual node addition still works
- Test error handling for kubectl failures

## Security Considerations
- SSH key must be properly configured for passwordless access
- Default paths should be appropriate for the environment
- Consider adding validation for discovered nodes before adding
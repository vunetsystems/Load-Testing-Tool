// Binary Control Module
class BinaryControl {
    constructor(manager) {
        this.manager = manager;
    }

    openBinaryControlModal() {
        console.log('Opening binary control modal');
        const modal = this.manager.elements.binaryControlModal;

        if (!modal) {
            console.error('Binary control modal not found!');
            return;
        }

        modal.classList.remove('hidden');
        this.populateBinaryNodeSelect();
        this.refreshAllBinaryStatuses();
    }

    closeBinaryControlModal() {
        console.log('Closing binary control modal');
        const modal = this.manager.elements.binaryControlModal;

        if (!modal) {
            console.error('Binary control modal not found!');
            return;
        }

        modal.classList.add('hidden');
        this.clearBinaryForm();
    }

    populateBinaryNodeSelect() {
        const nodeSelect = this.manager.elements.binaryNodeSelect;
        if (!nodeSelect) return;

        console.log('Populating binary node select with nodeData:', this.manager.nodeData);

        // Clear existing options except the first one
        while (nodeSelect.children.length > 1) {
            nodeSelect.removeChild(nodeSelect.lastChild);
        }

        // Add enabled nodes (both active and error status nodes are enabled)
        let addedNodes = 0;
        Object.keys(this.manager.nodeData).forEach(nodeId => {
            const node = this.manager.nodeData[nodeId];
            console.log(`Checking node ${nodeId}: status=${node.status}, enabled=${node.status !== 'inactive'}`);
            if (node.status === 'active' || node.status === 'error') {
                const option = document.createElement('option');
                option.value = nodeId;
                option.textContent = nodeId;
                nodeSelect.appendChild(option);
                addedNodes++;
                console.log(`Added node ${nodeId} to binary select`);
            }
        });

        console.log(`Total nodes added to binary select: ${addedNodes}`);
    }

    async refreshAllBinaryStatuses() {
        try {
            const response = await this.manager.callAPI('/api/binary/status');
            if (response.success && response.data) {
                this.displayAllBinaryStatuses(response.data);
            }
        } catch (error) {
            console.error('Error refreshing all binary statuses:', error);
        }
    }

    displayAllBinaryStatuses(statuses) {
        const tbody = this.manager.elements.binaryAllStatusBody;
        tbody.innerHTML = '';

        if (statuses.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="4" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No binary status data available</td>';
            tbody.appendChild(row);
            return;
        }

        statuses.forEach(status => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const statusBadge = status.status === 'running'
                ? '<div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark"><span class="h-2 w-2 rounded-full bg-success"></span>Running</div>'
                : status.status === 'stopped'
                    ? '<div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark"><span class="h-2 w-2 rounded-full bg-danger"></span>Stopped</div>'
                    : '<div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark"><span class="h-2 w-2 rounded-full bg-warning"></span>' + status.status + '</div>';

            row.innerHTML = `
                <td class="p-3 font-medium">${status.nodeName}</td>
                <td class="p-3">${statusBadge}</td>
                <td class="p-3">${status.pid || '-'}</td>
                <td class="p-3">${status.lastChecked || '-'}</td>
            `;

            tbody.appendChild(row);
        });
    }

    async onBinaryNodeSelectChange() {
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        if (selectedNode) {
            await this.showBinaryPathInfo(selectedNode);
            await this.refreshBinaryStatus();
        } else {
            this.hideBinaryPathInfo();
            this.hideBinaryStatusDisplay();
        }
    }

    async showBinaryPathInfo(selectedNode) {
        try {
            // Get node configuration to show binary path
            const nodesResponse = await this.manager.callAPI('/api/nodes');
            if (nodesResponse.success && nodesResponse.data) {
                const nodeData = nodesResponse.data.find(node => node.name === selectedNode);
                if (nodeData) {
                    const binaryPath = `${nodeData.binary_dir}/finalvudatasim`;
                    this.manager.elements.binaryPathDisplay.textContent = binaryPath;
                    this.manager.elements.binaryNameDisplay.textContent = 'finalvudatasim';
                    this.manager.elements.binaryPathInfo.classList.remove('hidden');

                    // Also show in SSH output
                    this.addSshOutput(`=== Node Selected: ${selectedNode} ===`);
                    this.addSshOutput(`Binary Directory: ${nodeData.binary_dir}`);
                    this.addSshOutput(`Binary Path: ${binaryPath}`);
                    this.addSshOutput(`Configuration: Host=${nodeData.host}, User=${nodeData.user}`);
                }
            }
        } catch (error) {
            console.error('Error getting node configuration:', error);
        }
    }

    hideBinaryPathInfo() {
        this.manager.elements.binaryPathInfo.classList.add('hidden');
    }

    async refreshBinaryStatus() {
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        if (!selectedNode) {
            this.manager.showNotification('Please select a node first', 'warning');
            return;
        }

        try {
            const response = await this.manager.callAPI(`/api/binary/status/${selectedNode}`);
            if (response.success && response.data) {
                this.displayBinaryStatus(response.data);
            }
        } catch (error) {
            console.error('Error refreshing binary status:', error);
            this.manager.showNotification('Failed to refresh binary status: ' + error.message, 'error');
        }
    }

    displayBinaryStatus(status) {
        this.manager.elements.binaryStatusDisplay.classList.remove('hidden');
        this.manager.elements.binaryCurrentStatus.textContent = status.status;
        this.manager.elements.binaryCurrentPid.textContent = status.pid || '-';
        this.manager.elements.binaryCurrentStartTime.textContent = status.startTime || '-';
        this.manager.elements.binaryCurrentLastChecked.textContent = status.lastChecked || '-';
        this.manager.elements.binaryCurrentProcessInfo.textContent = status.processInfo || '-';

        // Update button states based on status
        this.updateBinaryControlButtons(status.status);

        // Show additional binary path information if available
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        if (selectedNode) {
            this.addSshOutput(`=== Status Check for Node: ${selectedNode} ===`);
            this.addSshOutput(`Status: ${status.status} | PID: ${status.pid || 'N/A'} | Last Checked: ${status.lastChecked || 'N/A'}`);
        }
    }

    updateBinaryControlButtons(status) {
        const startBtn = this.manager.elements.startBinaryBtn;
        const stopBtn = this.manager.elements.stopBinaryBtn;

        if (status === 'running') {
            startBtn.disabled = true;
            stopBtn.disabled = false;
        } else if (status === 'stopped') {
            startBtn.disabled = false;
            stopBtn.disabled = true;
        } else {
            startBtn.disabled = true;
            stopBtn.disabled = true;
        }
    }

    hideBinaryStatusDisplay() {
        this.manager.elements.binaryStatusDisplay.classList.add('hidden');
        this.manager.elements.startBinaryBtn.disabled = true;
        this.manager.elements.stopBinaryBtn.disabled = true;
    }

    async startBinary() {
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        const timeout = parseInt(this.manager.elements.binaryTimeoutInput.value) || 30;

        if (!selectedNode) {
            this.manager.showNotification('Please select a node first', 'warning');
            return;
        }

        this.setBinaryButtonLoading('start', true);

        try {
            this.addSshOutput(`=== Starting Binary on Node: ${selectedNode} ===`);
            this.addSshOutput(`Checking binary existence...`);

            const response = await this.manager.callAPI(`/api/binary/start/${selectedNode}?timeout=${timeout}`, 'POST');

            if (response.success) {
                this.addSshOutput(`✓ Binary start command sent successfully`);
                if (response.data) {
                    if (response.data.binaryPath) {
                        this.addSshOutput(`✓ Binary path: ${response.data.binaryPath}`);
                    }
                    if (response.data.initialOutput) {
                        this.addSshOutput(`=== Initial Binary Output ===`);
                        this.addSshOutput(response.data.initialOutput);
                    }
                    if (response.data.warning === 'ALREADY_RUNNING') {
                        this.addSshOutput(`⚠ Warning: Binary was already running`);
                        if (response.data.currentOutput) {
                            this.addSshOutput(`=== Current Binary Output ===`);
                            this.addSshOutput(response.data.currentOutput);
                        }
                    }
                }
                this.manager.showNotification(`Binary start initiated on node ${selectedNode}`, 'success');

                // Refresh status after a delay
                setTimeout(() => {
                    this.refreshBinaryStatus();
                    this.refreshAllBinaryStatuses();
                }, 3000);
            } else {
                // Handle specific error cases
                if (response.data && response.data.error === 'BINARY_NOT_FOUND') {
                    this.addSshOutput(`✗ ERROR: Binary not found at ${response.data.binaryPath}`);
                    this.manager.showNotification(`Binary not found on node ${selectedNode}. Please check deployment.`, 'error');
                } else {
                    throw new Error(response.message || 'Failed to start binary');
                }
            }
        } catch (error) {
            console.error('Error starting binary:', error);
            this.manager.showNotification('Failed to start binary: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        } finally {
            this.setBinaryButtonLoading('start', false);
        }
    }

    async stopBinary() {
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        const timeout = parseInt(this.manager.elements.binaryTimeoutInput.value) || 30;

        if (!selectedNode) {
            this.manager.showNotification('Please select a node first', 'warning');
            return;
        }

        this.setBinaryButtonLoading('stop', true);

        try {
            this.addSshOutput(`=== Stopping Binary on Node: ${selectedNode} ===`);
            this.addSshOutput(`Timeout: ${timeout} seconds`);
            this.addSshOutput(`Executing stop command...`);

            const response = await this.manager.callAPI(`/api/binary/stop/${selectedNode}?timeout=${timeout}`, 'POST');
            if (response.success) {
                this.addSshOutput(`✓ Binary stop command sent successfully`);
                if (response.data && response.data.previousPID) {
                    this.addSshOutput(`✓ Previous PID: ${response.data.previousPID}`);
                }
                this.manager.showNotification(`Binary stop initiated on node ${selectedNode}`, 'success');

                // Refresh status after a delay
                setTimeout(() => {
                    this.refreshBinaryStatus();
                    this.refreshAllBinaryStatuses();
                }, 3000);
            } else {
                throw new Error(response.message || 'Failed to stop binary');
            }
        } catch (error) {
            console.error('Error stopping binary:', error);
            this.manager.showNotification('Failed to stop binary: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        } finally {
            this.setBinaryButtonLoading('stop', false);
        }
    }

    // Add method to get current binary output
    async getBinaryOutput() {
        const selectedNode = this.manager.elements.binaryNodeSelect.value;
        if (!selectedNode) {
            this.manager.showNotification('Please select a node first', 'warning');
            return;
        }

        try {
            this.addSshOutput(`=== Getting Current Output for Node: ${selectedNode} ===`);

            // Get node configuration to find log file path
            const nodesResponse = await this.manager.callAPI('/api/nodes');
            if (nodesResponse.success && nodesResponse.data) {
                const nodeData = nodesResponse.data.find(node => node.name === selectedNode);
                if (nodeData) {
                    const logFile = `${nodeData.binary_dir}/binary_output.log`;
                    this.addSshOutput(`Log file location: ${logFile}`);

                    // Try to get current output via a new API endpoint (would need backend implementation)
                    // For now, simulate by showing the log file content
                    this.addSshOutput(`Attempting to read current output...`);

                    // This is a placeholder - in a real implementation, you'd have an API endpoint
                    // that reads the log file and returns its content
                    this.addSshOutput(`Note: Real-time output reading requires backend API implementation`);
                    this.addSshOutput(`Log file would be tailed from: ${logFile}`);
                    this.manager.showNotification('Output retrieval requires backend API implementation', 'info');
                }
            }
        } catch (error) {
            console.error('Error getting binary output:', error);
            this.manager.showNotification('Failed to get binary output: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        }
    }

    setBinaryButtonLoading(action, loading) {
        const startBtn = this.manager.elements.startBinaryBtn;
        const stopBtn = this.manager.elements.stopBinaryBtn;

        if (action === 'start') {
            if (loading) {
                startBtn.disabled = true;
                startBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Starting...</span>';
            } else {
                startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start Binary</span>';
            }
        } else if (action === 'stop') {
            if (loading) {
                stopBtn.disabled = true;
                stopBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Stopping...</span>';
            } else {
                stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop Binary</span>';
            }
        }
    }

    addSshOutput(message) {
        const outputContainer = this.manager.elements.binarySshOutput;
        if (!outputContainer) return;

        const timestamp = new Date().toLocaleTimeString();
        const outputLine = document.createElement('div');
        outputLine.className = 'text-text-secondary-dark';
        outputLine.innerHTML = `<span class="text-sky-400">[${timestamp}]</span> ${message}`;

        outputContainer.appendChild(outputLine);

        // Auto-scroll to bottom
        outputContainer.scrollTop = outputContainer.scrollHeight;

        // Keep only last 50 lines
        while (outputContainer.children.length > 50) {
            outputContainer.removeChild(outputContainer.firstChild);
        }
    }

    clearBinaryForm() {
        this.manager.elements.binaryNodeSelect.value = '';
        this.hideBinaryPathInfo();
        this.hideBinaryStatusDisplay();
        this.manager.elements.binaryTimeoutInput.value = '30';
        this.manager.elements.binarySshOutput.innerHTML = '<div class="text-text-secondary-light dark:text-text-secondary-dark">SSH output will appear here...</div>';
    }
}
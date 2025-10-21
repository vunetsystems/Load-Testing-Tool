// Binary Control Module
class BinaryControl {
    constructor(manager) {
        this.manager = manager;
    }

    async openBinaryControlModal() {
        console.log('Opening binary control modal');
        const modal = this.manager.elements.binaryControlModal;

        if (!modal) {
            console.error('Binary control modal not found!');
            return;
        }

        modal.classList.remove('hidden');

        // Clear previous SSH output and add opening message
        const sshOutput = this.manager.elements.binarySshOutput;
        if (sshOutput) {
            sshOutput.innerHTML = '';
            this.addSshOutput('=== Binary Control Modal Opened ===');
            this.addSshOutput('Loading nodes for selection...');
        }

        try {
            await this.populateBinaryNodeSelect();
        } catch (error) {
            console.error('Error in populateBinaryNodeSelect:', error);
            this.addSshOutput(`✗ ERROR in populateBinaryNodeSelect: ${error.message}`);
        }

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

    async populateBinaryNodeSelect() {
        console.log('populateBinaryNodeSelect called');
        this.addSshOutput('populateBinaryNodeSelect function started');

        const nodeSelect = this.manager.elements.binaryNodeSelect;
        if (!nodeSelect) {
            console.error('Binary node select element not found!');
            this.addSshOutput('✗ ERROR: Binary node select element not found!');
            return;
        }
        this.addSshOutput('✓ Found binary node select element');

        console.log('Populating binary node select with nodeData:', this.manager.nodeData);

        // Clear existing options except the first one
        const initialCount = nodeSelect.children.length;
        while (nodeSelect.children.length > 1) {
            nodeSelect.removeChild(nodeSelect.lastChild);
        }
        this.addSshOutput(`Cleared ${initialCount - 1} existing options from dropdown`);

        let nodesData = [];

        // If nodeData is empty or not populated, load from API
        if (!this.manager.nodeData || Object.keys(this.manager.nodeData).length === 0) {
            console.log('nodeData is empty, loading nodes from API...');
            this.addSshOutput('nodeData is empty, loading nodes from API...');
            try {
                this.addSshOutput('Calling /api/nodes endpoint...');
                const response = await this.manager.callAPI('/api/nodes');
                console.log('API Response:', response);
                this.addSshOutput(`API response received: ${JSON.stringify(response)}`);

                if (response.success && response.data) {
                    nodesData = response.data;
                    console.log('Loaded nodes from API:', nodesData);
                    this.addSshOutput(`✓ API returned ${nodesData.length} nodes`);
                    this.addSshOutput(`Raw API data: ${JSON.stringify(nodesData)}`);
                } else {
                    console.error('Failed to load nodes from API:', response);
                    this.addSshOutput(`✗ ERROR: API failed - ${response.message || 'Unknown error'}`);
                    return;
                }
            } catch (error) {
                console.error('Error loading nodes from API:', error);
                this.addSshOutput(`✗ ERROR: ${error.message}`);
                return;
            }
        } else {
            // Convert nodeData format to API format for consistency
            nodesData = Object.keys(this.manager.nodeData).map(nodeId => {
                const node = this.manager.nodeData[nodeId];
                return {
                    name: nodeId,
                    host: node.host || '',
                    enabled: node.status === 'active' || node.status === 'error'
                };
            });
            this.addSshOutput(`Using existing nodeData with ${nodesData.length} nodes`);
        }

        // Add enabled nodes
        let addedNodes = 0;
        this.addSshOutput(`Processing ${nodesData.length} nodes for dropdown...`);
        nodesData.forEach((node, index) => {
            console.log(`Node ${index + 1}:`, node);
            this.addSshOutput(`Node ${index + 1}: ${JSON.stringify(node)}`);
            console.log(`Checking node ${node.name}: enabled=${node.enabled}`);
            if (node.enabled) {
                const option = document.createElement('option');
                option.value = node.name;
                option.textContent = node.name;
                nodeSelect.appendChild(option);
                addedNodes++;
                console.log(`Added node ${node.name} to binary select`);
                this.addSshOutput(`✓ Added enabled node: ${node.name}`);
            } else {
                this.addSshOutput(`- Skipped disabled node: ${node.name}`);
            }
        });

        console.log(`Total nodes added to binary select: ${addedNodes}`);
        this.addSshOutput(`Total nodes added to binary select: ${addedNodes}`);

        // Final debug output
        this.addSshOutput(`=== FINAL DEBUG SUMMARY ===`);
        this.addSshOutput(`Total nodes processed: ${nodesData.length}`);
        this.addSshOutput(`Enabled nodes added: ${addedNodes}`);
        this.addSshOutput(`Dropdown options count: ${nodeSelect.children.length}`);

        if (addedNodes === 0) {
            this.addSshOutput('⚠ WARNING: No enabled nodes found!');
            this.addSshOutput('This means either:');
            this.addSshOutput('1. No nodes are configured');
            this.addSshOutput('2. All nodes are disabled');
            this.addSshOutput('3. Node data format is incorrect');
        } else {
            this.addSshOutput(`✓ Successfully loaded ${addedNodes} nodes for selection`);
        }
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
// Dashboard Management Module
class DashboardManager {
    constructor(manager) {
        this.manager = manager;
        this.lastSshStatusUpdate = 0;
        this.sshStatuses = {};
        this.clusterMetrics = {};
    }

    async loadNodes() {
        try {
            console.log('Loading nodes...');
            // Load nodes from the integrated API
            const response = await this.manager.callAPI('/api/dashboard');
            console.log('Dashboard API response:', response);
            if (response.success && response.data && response.data.nodeData && Object.keys(response.data.nodeData).length > 0) {

                this.manager.nodeData = response.data.nodeData;
                console.log('Loaded nodeData from dashboard API:', this.manager.nodeData);

                // Load host information from nodes API to ensure we have host field for metrics matching
                const nodesResponse = await this.manager.callAPI('/api/nodes');
                if (nodesResponse.success && nodesResponse.data) {
                    nodesResponse.data.forEach(node => {
                        if (this.manager.nodeData[node.name]) {
                            this.manager.nodeData[node.name].host = node.host;
                            console.log(`Added host ${node.host} to node ${node.name}`);
                        }
                    });
                }

                this.updateDashboardDisplay();
                this.manager.updateNodeStatusIndicators();
                this.manager.populateNodeFilters();
            } else {
                console.log('Dashboard API did not return nodeData, trying nodes API...');
                // Fallback: load from nodes API if dashboard doesn't have nodeData
                const nodesResponse = await this.manager.callAPI('/api/nodes');
                console.log('Nodes API response:', nodesResponse);
                if (nodesResponse.success && nodesResponse.data) {
                    // Convert API data to nodeData format
                    this.manager.nodeData = {};
                    nodesResponse.data.forEach(node => {
                        const nodeId = node.name;
                        this.manager.nodeData[nodeId] = {
                            cpu: 0,           // Will be updated with real data
                            memory: 0,        // Will be updated with real data
                            totalCpu: 8.0,
                            totalMemory: 8.0,
                            status: node.enabled ? 'active' : 'inactive',
                            host: node.host  // Store the host IP for matching with metrics target

                        };
                        console.log(`Converted node ${nodeId} with host ${node.host} and status ${this.manager.nodeData[nodeId].status}`);
                    });
                    console.log('Final nodeData after conversion:', this.manager.nodeData);

                    this.updateDashboardDisplay();
                    this.manager.updateNodeStatusIndicators();
                    this.manager.populateNodeFilters();
                }
            }
        } catch (error) {
            console.error('Error loading nodes:', error);
        }
    }

    async updateDashboardDisplay() {
        // Update cluster table dynamically
        const tbody = document.getElementById('cluster-table-body');
        tbody.innerHTML = '';

        // Fetch SSH status for enabled nodes (only once per hour)
        const now = Date.now();
        const oneHour = 60 * 60 * 1000; // 1 hour in milliseconds

        if (now - this.lastSshStatusUpdate > oneHour || Object.keys(this.sshStatuses).length === 0) {
            try {
                console.log('Fetching SSH status (hourly update)...');
                const sshResponse = await this.manager.callAPI('/api/ssh/status');
                if (sshResponse.success && sshResponse.data) {
                    this.sshStatuses = {};
                    sshResponse.data.forEach(status => {
                        this.sshStatuses[status.nodeName] = status;
                    });
                    this.lastSshStatusUpdate = now;
                    console.log('SSH status updated:', this.sshStatuses);
                }
            } catch (error) {
                console.error('Error fetching SSH status:', error);
            }
        }

        const sshStatuses = this.sshStatuses;

        Object.keys(this.manager.nodeData).forEach(nodeId => {
            const node = this.manager.nodeData[nodeId];
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            // Check if node is disabled (not enabled in the configuration)
            const isNodeDisabled = node.status === 'inactive';
            if (isNodeDisabled) {
                row.classList.add('opacity-60');
            }

            // Calculate CPU usage - use real metrics if available
            let cpuDisplay = '';
            const realMetrics = this.clusterMetrics ? Object.values(this.clusterMetrics).find(m => m.target === node.host) : null;
            if (realMetrics) {
                const availableCpu = realMetrics.cpu_cores * 0.1; // Show 10% as available for example
                cpuDisplay = `${availableCpu.toFixed(1)} / ${realMetrics.cpu_cores.toFixed(1)} cores`;
            } else {
                // Fallback to old calculation
                cpuDisplay = `${(node.totalCpu - (node.totalCpu * node.cpu / 100)).toFixed(1)} / ${node.totalCpu} cores`;
            }

            // Calculate memory usage - use real metrics if available
            let memoryDisplay = '';
            if (realMetrics) {
                memoryDisplay = `${realMetrics.used_memory_gb.toFixed(1)} / ${realMetrics.total_memory_gb.toFixed(1)} GB`;
            } else {
                // Fallback to old calculation
                const usedMemory = node.totalMemory * (node.memory / 100);
                memoryDisplay = `${usedMemory.toFixed(1)} / ${node.totalMemory} GB`;
            }

            // Determine status display based on SSH connectivity for enabled nodes
            let statusDisplay = '';
            if (!isNodeDisabled) {
                const sshStatus = sshStatuses[nodeId];
                if (sshStatus) {
                    const timestamp = new Date(sshStatus.lastChecked).toLocaleString();
                    if (sshStatus.status === 'connected') {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark">
                                    <span class="h-2 w-2 rounded-full bg-success"></span>SSH Connected
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    } else if (sshStatus.status === 'disconnected') {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark">
                                    <span class="h-2 w-2 rounded-full bg-danger"></span>SSH Disconnected
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    } else {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark">
                                    <span class="h-2 w-2 rounded-full bg-warning"></span>SSH ${sshStatus.status}
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    }
                } else {
                    statusDisplay = `
                        <div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark">
                            <span class="h-2 w-2 rounded-full bg-warning"></span>Checking SSH...
                        </div>
                    `;
                }
            } else {
                statusDisplay = `
                    <div class="inline-flex items-center gap-2 rounded-full bg-gray-500/20 dark:bg-gray-400/20 px-3 py-1 text-xs font-medium text-gray-500 dark:text-gray-400">
                        <span class="h-2 w-2 rounded-full bg-gray-500"></span>Disabled
                    </div>
                `;
            }

            row.innerHTML = `
                <td class="p-4 font-medium">${nodeId}</td>
                <td class="p-4">${statusDisplay}</td>
                <td class="p-4 text-right number-animate" data-field="cpu">${cpuDisplay}</td>
                <td class="p-4 text-right number-animate" data-field="memory">${memoryDisplay}</td>
            `;

            tbody.appendChild(row);
        });

        // Calculate real-time CPU/Memory data for active and error nodes (nodes that are enabled but may have connection issues)
        // Calculate real-time CPU/Memory data for active and error nodes
        const displayableNodes = Object.values(this.manager.nodeData).filter(node => node.status === 'active' || node.status === 'error');
        if (displayableNodes.length > 0) {
            // Use real metrics if available, otherwise fall back to defaults
            let totalRealCpu = 0;
            let totalRealMemory = 0;
            let totalUsedMemory = 0;
            let nodeCount = 0;

            displayableNodes.forEach(node => {
                const realMetrics = this.clusterMetrics[node.nodeId || node.name];
                if (realMetrics) {
                    totalRealCpu += realMetrics.cpu_cores;
                    totalRealMemory += realMetrics.total_memory_gb;
                    totalUsedMemory += realMetrics.used_memory_gb;
                    nodeCount++;
                }
            });

            if (nodeCount > 0) {
                // Use real metrics
                const avgRealCpu = totalRealCpu / nodeCount;
                const avgRealMemory = totalRealMemory / nodeCount;
                const avgUsedMemory = totalUsedMemory / nodeCount;
                const availableCpu = avgRealCpu * 0.1; // Show 10% usage as example

                this.manager.elements.cpuMemoryValue.textContent = `${availableCpu.toFixed(1)}/${avgRealCpu.toFixed(1)} cores / ${avgUsedMemory.toFixed(1)}/${avgRealMemory.toFixed(1)} GB`;
            } else {
                // Fallback to old calculation
                const totalAvgCpu = displayableNodes.reduce((sum, node) => sum + node.totalCpu, 0) / displayableNodes.length;
                const totalAvgMemory = displayableNodes.reduce((sum, node) => sum + node.totalMemory, 0) / displayableNodes.length;
                const avgCpuUsage = displayableNodes.reduce((sum, node) => sum + node.cpu, 0) / displayableNodes.length;
                const avgMemoryUsage = displayableNodes.reduce((sum, node) => sum + node.memory, 0) / displayableNodes.length;
                const availableCpu = totalAvgCpu - (totalAvgCpu * avgCpuUsage / 100);
                const usedMemory = totalAvgMemory * (avgMemoryUsage / 100);

                this.manager.elements.cpuMemoryValue.textContent = `${availableCpu.toFixed(1)}/${totalAvgCpu.toFixed(1)} cores / ${usedMemory.toFixed(1)}/${totalAvgMemory.toFixed(1)} GB`;
            }

        }
    }

    updateClusterTableOnly() {
        // Update only the metrics columns in existing cluster table rows
        const tbody = document.getElementById('cluster-table-body');
        if (!tbody) return;

        const rows = tbody.querySelectorAll('tr');

        rows.forEach(row => {
            const nodeName = row.cells[0]?.textContent?.trim();
            const node = this.manager.nodeData[nodeName];

            if (node) {
                // Find real metrics for this node by matching host
                const realMetrics = this.clusterMetrics ?
                    Object.values(this.clusterMetrics).find(m => m.target === node.host) : null;

                // Update CPU column (index 2)
                if (row.cells[2] && realMetrics) {
                    const availableCpu = realMetrics.cpu_cores * 0.1; // Show 10% as available for example
                    row.cells[2].textContent = `${availableCpu.toFixed(1)} / ${realMetrics.cpu_cores.toFixed(1)} cores`;
                    row.cells[2].classList.add('number-animate');
                }

                // Update Memory column (index 3)
                if (row.cells[3] && realMetrics) {
                    row.cells[3].textContent = `${realMetrics.used_memory_gb.toFixed(1)} / ${realMetrics.total_memory_gb.toFixed(1)} GB`;
                    row.cells[3].classList.add('number-animate');
                }
            }
        });
    }

    async updateSSHStatuses() {
        // Fetch SSH status for enabled nodes (only once per hour)
        const now = Date.now();
        const oneHour = 60 * 60 * 1000; // 1 hour in milliseconds

        if (now - this.lastSshStatusUpdate > oneHour || Object.keys(this.sshStatuses).length === 0) {
            try {
                console.log('Fetching SSH status (hourly update)...');
                const sshResponse = await this.manager.callAPI('/api/ssh/status');
                if (sshResponse.success && sshResponse.data) {
                    this.sshStatuses = {};
                    sshResponse.data.forEach(status => {
                        this.sshStatuses[status.nodeName] = status;
                    });
                    this.lastSshStatusUpdate = now;
                    console.log('SSH status updated:', this.sshStatuses);
                }
            } catch (error) {
                console.error('Error fetching SSH status:', error);
            }
        }

        // Update only the status column in existing rows
        const tbody = document.getElementById('cluster-table-body');
        const rows = tbody.querySelectorAll('tr');

        rows.forEach(row => {
            const nodeName = row.cells[0]?.textContent?.trim();
            const node = this.manager.nodeData[nodeName];

            if (node && node.status !== 'inactive') { // Only update enabled nodes
                const sshStatus = this.sshStatuses[nodeName];
                let statusDisplay = '';

                if (sshStatus) {
                    const timestamp = new Date(sshStatus.lastChecked).toLocaleString();
                    if (sshStatus.status === 'connected') {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark">
                                    <span class="h-2 w-2 rounded-full bg-success"></span>SSH Connected
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    } else if (sshStatus.status === 'disconnected') {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark">
                                    <span class="h-2 w-2 rounded-full bg-danger"></span>SSH Disconnected
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    } else {
                        statusDisplay = `
                            <div class="flex flex-col items-start gap-1">
                                <div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark">
                                    <span class="h-2 w-2 rounded-full bg-warning"></span>SSH ${sshStatus.status}
                                </div>
                                <div class="text-xs text-text-secondary-light dark:text-text-secondary-dark">
                                    Last checked: ${timestamp}
                                </div>
                            </div>
                        `;
                    }
                } else {
                    statusDisplay = `
                        <div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark">
                            <span class="h-2 w-2 rounded-full bg-warning"></span>Checking SSH...
                        </div>
                    `;
                }

                // Update only the status cell (index 1)
                if (row.cells[1]) {
                    row.cells[1].innerHTML = statusDisplay;
                }
            }
        });
    }

    async fetchFinalVuDataSimMetrics() {
        try {
            // Directly fetch nodes from API instead of relying on loaded nodeData
            console.log('Fetching nodes from API...');
            const nodesResponse = await this.manager.callAPI('/api/nodes');
            if (!nodesResponse.success || !nodesResponse.data) {
                console.log('Failed to fetch nodes from API');
                this.displayFinalVuDataSimMetrics([]);
                return;
            }

            // Filter for enabled nodes
            const enabledNodes = nodesResponse.data.filter(node => node.enabled);
            console.log('Enabled nodes found:', enabledNodes.map(n => n.name));

            if (enabledNodes.length === 0) {
                console.log('No enabled nodes found for metrics');
                this.displayFinalVuDataSimMetrics([]);
                return;
            }

            const allMetrics = [];

            // Fetch metrics from each enabled node
            for (const node of enabledNodes) {
                try {
                    console.log(`Fetching metrics from node: ${node.name}`);

                    // Use proxy endpoint with node name parameter
                    const response = await this.manager.callAPI(`/api/proxy/metrics/${node.name}`);
                    console.log(`=== METRICS FROM ${node.name} ===`);
                    console.log('Full response:', response);

                    // The proxy endpoint returns the raw metrics data directly
                    if (response && typeof response === 'object' && response.nodeId) {
                        console.log(`Valid metrics data received from ${node.name}, adding to display...`);
                        // Add node name to the response for display
                        response.nodeName = node.name;
                        allMetrics.push(response);
                    } else {
                        console.log(`No valid metrics data from ${node.name}, adding error entry`);
                        // Add error entry for this node
                        allMetrics.push({
                            nodeId: node.name,
                            nodeName: node.name,
                            process: { running: false, pid: 0, start_time: 'N/A', cpu_percent: 0, mem_mb: 0, cmdline: 'N/A' },
                            system: { cpu_usage: 0, cpu_cores: 0, mem_total_mb: 0, mem_used_mb: 0, mem_free_mb: 0, disk_total_gb: 0, disk_used_gb: 0, disk_free_gb: 0, load_avg_1: 0, load_avg_5: 0, load_avg_15: 0, uptime: 'N/A' },
                            timestamp: new Date().toISOString(),
                            error: 'Metrics unavailable'
                        });
                    }
                } catch (nodeError) {
                    console.error(`Error fetching metrics from ${node.name}:`, nodeError);
                    // Add error entry for this node
                    allMetrics.push({
                        nodeId: node.name,
                        nodeName: node.name,
                        process: { running: false, pid: 0, start_time: 'N/A', cpu_percent: 0, mem_mb: 0, cmdline: 'N/A' },
                        system: { cpu_usage: 0, cpu_cores: 0, mem_total_mb: 0, mem_used_mb: 0, mem_free_mb: 0, disk_total_gb: 0, disk_used_gb: 0, disk_free_gb: 0, load_avg_1: 0, load_avg_5: 0, load_avg_15: 0, uptime: 'N/A' },
                        timestamp: new Date().toISOString(),
                        error: 'Connection failed'
                    });
                }
            }

            console.log('All metrics collected:', allMetrics);
            this.displayFinalVuDataSimMetrics(allMetrics);

        } catch (error) {
            console.error('Error in fetchFinalVuDataSimMetrics:', error);
            this.displayFinalVuDataSimMetrics([]);
        }
    }

    async updateVuDataSimButtonState() {
        const startBtn = this.manager.elements.startVuDataSimBtn;
        const stopBtn = this.manager.elements.stopVuDataSimBtn;

        if (!startBtn || !stopBtn) return;

        try {
            // Get the actual binary status from the backend using the same method as binary control
            const statusResponse = await this.manager.callAPI('/api/binary/status');
            if (statusResponse.success && statusResponse.data) {
                // Check if any node has status NOT "stopped" (i.e., running or any other status)
                const anyNodeRunning = statusResponse.data.some(status =>
                    status.status !== 'stopped'
                );

                if (anyNodeRunning) {
                    // Disable start button when any node is running
                    startBtn.disabled = true;
                    startBtn.innerHTML = '<span class="material-symbols-outlined">play_disabled</span><span>Start vuDataSim</span>';
                    startBtn.classList.add('opacity-50', 'cursor-not-allowed');
                    startBtn.classList.remove('hover:scale-105', 'active:scale-100');
                } else {
                    // Enable start button when all nodes are stopped
                    startBtn.disabled = false;
                    startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start vuDataSim</span>';
                    startBtn.classList.remove('opacity-50', 'cursor-not-allowed');
                    startBtn.classList.add('hover:scale-105', 'active:scale-100');
                }

                // Stop button is always enabled
                stopBtn.disabled = false;
                stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop vuDataSim</span>';
                stopBtn.classList.remove('opacity-50', 'cursor-not-allowed');
                stopBtn.classList.add('hover:scale-105', 'active:scale-100');

                // Update Configuration Sync Status step6
                this.updateConfigSyncStep6(anyNodeRunning);
            } else {
                console.error('Failed to get binary status:', statusResponse.message);
                // On error, disable start button to be safe
                startBtn.disabled = true;
                startBtn.innerHTML = '<span class="material-symbols-outlined">play_disabled</span><span>Start vuDataSim</span>';
                startBtn.classList.add('opacity-50', 'cursor-not-allowed');
                startBtn.classList.remove('hover:scale-105', 'active:scale-100');
                stopBtn.disabled = false;
                stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop vuDataSim</span>';
                stopBtn.classList.remove('opacity-50', 'cursor-not-allowed');
                stopBtn.classList.add('hover:scale-105', 'active:scale-100');
                this.updateConfigSyncStep6(false);
            }
        } catch (error) {
            console.error('Error fetching binary status:', error);
            // On error, disable start button to be safe
            startBtn.disabled = true;
            startBtn.innerHTML = '<span class="material-symbols-outlined">play_disabled</span><span>Start vuDataSim</span>';
            startBtn.classList.add('opacity-50', 'cursor-not-allowed');
            startBtn.classList.remove('hover:scale-105', 'active:scale-100');
            stopBtn.disabled = false;
            stopBtn.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop vuDataSim</span>';
            stopBtn.classList.remove('opacity-50', 'cursor-not-allowed');
            stopBtn.classList.add('hover:scale-105', 'active:scale-100');
            this.updateConfigSyncStep6(false);
        }
    }

    updateConfigSyncStep6(isRunning) {
        const step6Icon = this.manager.elements.step6Icon;
        const step6Text = this.manager.elements.step6Text;
        const step6Title = this.manager.elements.step6Title;
        const step6Container = this.manager.elements.step6Container;
        const step6Box = this.manager.elements.step6Box;

        if (!step6Icon || !step6Text || !step6Title || !step6Container || !step6Box) return;

        if (isRunning) {
            // Update step6 to show "Running"
            step6Icon.textContent = 'check_circle';
            step6Text.textContent = 'Running';
            step6Title.textContent = 'Start vuDataSim';
            step6Container.className = 'relative w-10 h-10 mx-auto flex items-center justify-center rounded-full bg-green-100 text-green-600';
            step6Box.className = 'mt-4 p-4 rounded-lg bg-green-50/50 border border-green-200 hover:shadow-md transition-shadow';
        } else {
            // Reset step6 to pending state
            step6Icon.textContent = 'schedule';
            step6Text.textContent = 'Pending';
            step6Title.textContent = 'Start vuDataSim';
            step6Container.className = 'relative w-10 h-10 mx-auto flex items-center justify-center rounded-full bg-slate-100 text-slate-400';
            step6Box.className = 'mt-4 p-4 rounded-lg bg-slate-50/50 border border-slate-200 hover:shadow-md transition-shadow';
        }
    }

    displayFinalVuDataSimMetrics(metricsArray) {
        const tbody = document.getElementById('finalvudatasim-metrics-body');
        if (!tbody) return;
        tbody.innerHTML = '';

        if (!Array.isArray(metricsArray) || metricsArray.length === 0) {
            // Show empty state
            const row = document.createElement('tr');
            row.innerHTML = `
                <td colspan="11" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No metrics available
                </td>
            `;
            tbody.appendChild(row);
            // Update button state based on actual binary status from backend
            this.updateVuDataSimButtonState();
            return;
        }

        // Check if any node has vuDataSim running
        let anyNodeRunning = false;

        // Display metrics for each node
        metricsArray.forEach(metrics => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            // Process metrics
            const process = metrics.process || {};
            const system = metrics.system || {};

            // Robust running detection
            const isRunning = process.running || (process.pid && process.pid > 0);
            if (isRunning) {
                anyNodeRunning = true;
            }

            // Format load average
            const loadAvg = system.load_avg_1 !== undefined ?
                `${system.load_avg_1.toFixed(2)}, ${system.load_avg_5.toFixed(2)}, ${system.load_avg_15.toFixed(2)}` : '-';

            // Format memory values
            const memUsedGB = system.mem_used_mb ? (system.mem_used_mb / 1024).toFixed(1) : '-';
            const memTotalGB = system.mem_total_mb ? (system.mem_total_mb / 1024).toFixed(1) : '-';

            row.innerHTML = `
                <td class="p-4 font-medium">${metrics.nodeName || metrics.nodeId || 'Unknown'}</td>
                <td class="p-4">${isRunning ? '<span class="text-success font-bold">Yes</span>' : '<span class="text-danger font-bold">No</span>'}</td>
                <td class="p-4">${process.pid && process.pid > 0 ? process.pid : '-'}</td>
                <td class="p-4">${process.start_time && process.start_time !== '' ? process.start_time : '-'}</td>
                <td class="p-4">${typeof process.cpu_percent === 'number' ? process.cpu_percent.toFixed(2) : '-'}</td>
                <td class="p-4">${typeof process.mem_mb === 'number' ? process.mem_mb.toFixed(2) : '-'}</td>
                <td class="p-4">${typeof system.cpu_usage === 'number' ? system.cpu_usage.toFixed(2) : '-'}</td>
                <td class="p-4">${memUsedGB}/${memTotalGB} GB</td>
                <td class="p-4">${typeof system.disk_used_gb === 'number' ? system.disk_used_gb.toFixed(1) : '-'}/${typeof system.disk_total_gb === 'number' ? system.disk_total_gb.toFixed(1) : '-'} GB</td>
                <td class="p-4">${loadAvg}</td>
                <td class="p-4">${system.uptime || '-'}</td>
            `;
            tbody.appendChild(row);
        });

        // Update button state based on actual binary status from backend
        this.updateVuDataSimButtonState();
    }
}
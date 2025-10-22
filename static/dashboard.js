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
            // Use proxy endpoint instead of direct call to avoid CORS
            const response = await this.manager.callAPI('/api/proxy/metrics');
            console.log('=== FINALVUDATASIM PROCESS METRICS ===');
            console.log('Full response:', response);
            console.log('Response data:', response.data);
            console.log('Response success:', response.success);
            // The proxy endpoint returns the raw metrics data directly, not wrapped in success/data structure
            if (response && response.running !== undefined) {
                console.log('Valid process metrics data received, displaying...');
                this.displayFinalVuDataSimMetrics(response);
            } else {
                console.log('No valid process metrics data in response');
                this.displayFinalVuDataSimMetrics({}); // Show empty row on error
            }
        } catch (error) {
            console.error('Error fetching finalvudatasim metrics:', error);
            this.displayFinalVuDataSimMetrics({}); // Show empty row on error
        }
    }

    displayFinalVuDataSimMetrics(metrics) {
        const tbody = document.getElementById('finalvudatasim-metrics-body');
        if (!tbody) return;
        tbody.innerHTML = '';
        // Robust running detection: if running is true OR pid is present and > 0
        const isRunning = metrics.running || (metrics.pid && metrics.pid > 0);
        const row = document.createElement('tr');
        row.innerHTML = `
            <td class="p-4">${isRunning ? '<span class="text-success font-bold">Yes</span>' : '<span class="text-danger font-bold">No</span>'}</td>
            <td class="p-4">${metrics.pid && metrics.pid > 0 ? metrics.pid : '-'}</td>
            <td class="p-4">${metrics.start_time ? metrics.start_time : '-'}</td>
            <td class="p-4">${typeof metrics.cpu_percent === 'number' ? metrics.cpu_percent.toFixed(2) : '-'}</td>
            <td class="p-4">${typeof metrics.mem_mb === 'number' ? metrics.mem_mb.toFixed(2) : '-'}</td>
            <td class="p-4">${metrics.cmdline ? metrics.cmdline : '-'}</td>
        `;
        tbody.appendChild(row);
    }
}
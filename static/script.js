// vuDataSim Cluster Manager - Frontend JavaScript
class VuDataSimManager {
    constructor() {
        console.log('VuDataSimManager constructor called');
        this.isSimulationRunning = false;
        this.apiBaseUrl = ''; // Backend API base URL (empty for same origin)
        this.wsConnection = null;
        this.nodes = {}; // Store node data
        // Add this property to the constructor
        this.clusterMetrics = {}; // Store real cluster metrics

        this.initializeComponents();
        this.bindEvents();
        this.loadNodes(); // Load nodes from API
        this.startRealTimeUpdates();
        this.populateNodeFilters(); // Populate filter dropdowns with real node names

        // Load o11y sources after DOM is ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                console.log('DOM loaded, about to load O11y sources...');
                this.loadO11ySources(); // Load available o11y sources
            });
        } else {
            console.log('DOM already loaded, loading O11y sources...');
            setTimeout(() => {
                this.loadO11ySources(); // Load available o11y sources
            }, 100);
        }
        console.log('VuDataSimManager initialization complete');
    }

    initializeComponents() {
        console.log('Initializing components...');
        // Cache DOM elements for better performance
        console.log('DOM elements for O11y sources:');
        console.log('- o11ySourcesContainer:', document.getElementById('o11y-sources-container'));
        console.log('- o11ySourcesDropdown:', document.getElementById('o11y-sources-dropdown'));
        console.log('- o11ySourcesOptions:', document.getElementById('o11y-sources-options'));
        console.log('- o11ySourcesList:', document.getElementById('o11y-sources-list'));
        this.elements = {
            syncBtn: document.getElementById('sync-btn'),
            logNodeFilter: document.getElementById('log-node'),
            logModuleFilter: document.getElementById('log-module'),
            logsContainer: document.getElementById('logs-container'),

            // O11y source management elements
            o11ySourcesContainer: document.getElementById('o11y-sources-container'),
            o11ySourcesDropdown: document.getElementById('o11y-sources-dropdown'),
            o11ySourcesOptions: document.getElementById('o11y-sources-options'),
            o11ySourcesList: document.getElementById('o11y-sources-list'),
            o11ySourcesSearch: document.getElementById('o11y-sources-search'),
            o11ySourcesPlaceholder: document.getElementById('o11y-sources-placeholder'),
            o11ySourcesSelected: document.getElementById('o11y-sources-selected'),
            o11ySourcesArrow: document.getElementById('o11y-sources-arrow'),
            o11ySourcesSelectAll: document.getElementById('o11y-sources-select-all'),
            o11ySourcesClearAll: document.getElementById('o11y-sources-clear-all'),
            o11ySourcesCount: document.getElementById('o11y-sources-count'),
            epsSelect: document.getElementById('eps-select'),
            syncConfigsBtn: document.getElementById('sync-configs-btn'),
            syncStatusContainer: document.getElementById('sync-status-container'),
            syncSuccessMessage: document.getElementById('sync-success-message'),
            syncErrorMessage: document.getElementById('sync-error-message'),

            // Node management elements
            nodeManagementBtn: document.getElementById('node-management-btn'),
            nodeManagementModal: document.getElementById('node-management-modal'),
            modalBackdrop: document.getElementById('modal-backdrop'),
            closeNodeModal: document.getElementById('close-node-modal'),
            nodeName: document.getElementById('node-name'),
            nodeHost: document.getElementById('node-host'),
            nodeUser: document.getElementById('node-user'),
            nodeKeypath: document.getElementById('node-keypath'),
            nodeConfdir: document.getElementById('node-confdir'),
            nodeBindir: document.getElementById('node-bindir'),
            nodeDescription: document.getElementById('node-description'),
            nodeEnabled: document.getElementById('node-enabled'),
            addNodeBtn: document.getElementById('add-node-btn'),
            nodesTableBody: document.getElementById('nodes-table-body'),

            // Binary control elements
            binaryControlBtn: document.getElementById('binary-control-btn'),
            binaryControlModal: document.getElementById('binary-control-modal'),
            binaryModalBackdrop: document.getElementById('binary-modal-backdrop'),
            closeBinaryModal: document.getElementById('close-binary-modal'),
            binaryNodeSelect: document.getElementById('binary-node-select'),
            binaryPathInfo: document.getElementById('binary-path-info'),
            binaryPathDisplay: document.getElementById('binary-path-display'),
            binaryNameDisplay: document.getElementById('binary-name-display'),
            binaryStatusDisplay: document.getElementById('binary-status-display'),
            binaryCurrentStatus: document.getElementById('binary-current-status'),
            binaryCurrentPid: document.getElementById('binary-current-pid'),
            binaryCurrentStartTime: document.getElementById('binary-current-start-time'),
            binaryCurrentLastChecked: document.getElementById('binary-current-last-checked'),
            binaryCurrentProcessInfo: document.getElementById('binary-current-process-info'),
            refreshBinaryStatusBtn: document.getElementById('refresh-binary-status-btn'),
            startBinaryBtn: document.getElementById('start-binary-btn'),
            stopBinaryBtn: document.getElementById('stop-binary-btn'),
            binaryTimeoutInput: document.getElementById('binary-timeout-input'),
            binarySshOutput: document.getElementById('binary-ssh-output'),
            binaryAllStatusBody: document.getElementById('binary-all-status-body'),

            // Dashboard elements
            nodeStatusIndicatorsContainer: document.getElementById('node-status-indicators'),

            // Chart value elements
            cpuMemoryValue: document.getElementById('cpu-memory-value'),

            // ClickHouse metrics elements
            clickHouseMetricsBtn: document.getElementById('clickhouse-metrics-btn'),
            clickHouseMetricsModal: document.getElementById('clickhouse-metrics-modal'),
            clickHouseModalBackdrop: document.getElementById('clickhouse-modal-backdrop'),
            closeClickHouseModal: document.getElementById('close-clickhouse-modal'),
            refreshClickHouseMetricsBtn: document.getElementById('refresh-clickhouse-metrics-btn'),
            clickHouseStatus: document.getElementById('clickhouse-status'),
            clickHouseLastUpdate: document.getElementById('clickhouse-last-update'),
            kafkaMetricsTable: document.getElementById('kafka-metrics-table'),
            systemMetricsTable: document.getElementById('system-metrics-table'),
            databaseMetricsTable: document.getElementById('database-metrics-table'),
            containerMetricsTable: document.getElementById('container-metrics-table'),
            podResourceMetricsTable: document.getElementById('pod-resource-metrics-table'),
            podStatusMetricsTable: document.getElementById('pod-status-metrics-table'),

            // Real-time status
        };

        // Initialize node data (will be loaded from API with real data only)
        this.nodeData = {};

        // SSH status cache (updated hourly)
        this.sshStatuses = {};
        this.lastSshStatusUpdate = 0;

        // Legacy property for backward compatibility (browser cache)
        this.nodeManagementSection = null;

        // Real-time log entries (will be loaded from API)
        this.logEntries = [];
        this.lastLogUpdate = 0;
    }

    bindEvents() {
        // Button event listeners
        this.elements.syncBtn?.addEventListener('click', () => this.refreshRealData());

        // O11y source management event listeners
        this.elements.syncConfigsBtn?.addEventListener('click', () => this.syncConfigs());

        // New custom multi-select event listeners
        this.elements.o11ySourcesDropdown?.addEventListener('click', () => this.toggleO11ySourcesDropdown());
        this.elements.o11ySourcesSearch?.addEventListener('input', (e) => this.filterO11ySources(e.target.value));
        this.elements.o11ySourcesSelectAll?.addEventListener('click', () => this.selectAllO11ySources());
        this.elements.o11ySourcesClearAll?.addEventListener('click', () => this.clearAllO11ySources());

        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!this.elements.o11ySourcesContainer?.contains(e.target)) {
                this.closeO11ySourcesDropdown();
            }
        });

        // Node management event listeners
        console.log('Node Management Button element:', this.elements.nodeManagementBtn);
        console.log('Node Management Modal element:', this.elements.nodeManagementModal);

        if (this.elements.nodeManagementBtn) {
            this.elements.nodeManagementBtn.addEventListener('click', () => {
                console.log('Node Management button clicked!');
                this.openNodeManagementModal();
            });
        } else {
            console.error('Node Management button not found!');
        }

        // Binary control event listeners
        console.log('Binary Control Button element:', this.elements.binaryControlBtn);
        console.log('Binary Control Modal element:', this.elements.binaryControlModal);

        if (this.elements.binaryControlBtn) {
            this.elements.binaryControlBtn.addEventListener('click', () => {
                console.log('Binary Control button clicked!');
                this.openBinaryControlModal();
            });
        } else {
            console.error('Binary Control button not found!');
        }

        // ClickHouse metrics event listeners
        console.log('ClickHouse Metrics Button element:', this.elements.clickHouseMetricsBtn);
        console.log('ClickHouse Metrics Modal element:', this.elements.clickHouseMetricsModal);

        if (this.elements.clickHouseMetricsBtn) {
            this.elements.clickHouseMetricsBtn.addEventListener('click', () => {
                console.log('ClickHouse Metrics button clicked!');
                this.openClickHouseMetricsModal();
            });
        } else {
            console.error('ClickHouse Metrics button not found!');
        }

        // Modal event listeners
        this.elements.closeNodeModal?.addEventListener('click', () => this.closeNodeManagementModal());
        this.elements.modalBackdrop?.addEventListener('click', () => this.closeNodeManagementModal());

        // Binary control modal event listeners
        this.elements.closeBinaryModal?.addEventListener('click', () => this.closeBinaryControlModal());
        this.elements.binaryModalBackdrop?.addEventListener('click', () => this.closeBinaryControlModal());

        // ClickHouse metrics modal event listeners
        this.elements.closeClickHouseModal?.addEventListener('click', () => this.closeClickHouseMetricsModal());
        this.elements.clickHouseModalBackdrop?.addEventListener('click', () => this.closeClickHouseMetricsModal());
        this.elements.refreshClickHouseMetricsBtn?.addEventListener('click', () => this.refreshClickHouseMetrics());

        // Binary control action listeners
        this.elements.refreshBinaryStatusBtn?.addEventListener('click', () => this.refreshBinaryStatus());
        this.elements.startBinaryBtn?.addEventListener('click', () => this.startBinary());
        this.elements.stopBinaryBtn?.addEventListener('click', () => this.stopBinary());
        this.elements.binaryNodeSelect?.addEventListener('change', () => this.onBinaryNodeSelectChange());

        // Add get binary output button listener
        const getOutputBtn = document.getElementById('get-binary-output-btn');
        if (getOutputBtn) {
            getOutputBtn.addEventListener('click', () => this.getBinaryOutput());
        }

        // Close modal on Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (!this.elements.nodeManagementModal.classList.contains('hidden')) {
                    this.closeNodeManagementModal();
                }
                if (!this.elements.binaryControlModal.classList.contains('hidden')) {
                    this.closeBinaryControlModal();
                }
                if (!this.elements.clickHouseMetricsModal.classList.contains('hidden')) {
                    this.closeClickHouseMetricsModal();
                }
            }
        });

        this.elements.addNodeBtn?.addEventListener('click', () => this.addNode());

        // Edit node form (initially hidden)
        this.isEditMode = false;
        this.editNodeName = null;


        // Log filtering
        this.elements.logNodeFilter?.addEventListener('change', () => this.filterLogs());
        this.elements.logModuleFilter?.addEventListener('change', () => this.filterLogs());

        // Real-time updates - Update every 3 seconds with real data
        setInterval(() => {
            this.updateMetrics();
            this.updateNodeStatusIndicators(); // Ensure status dots stay in sync
            // Note: updateDashboardDisplay() is now called hourly for SSH status
        }, 3000);

        // Cluster metrics updates - Update every 30 seconds
        setInterval(async () => {
            try {
                const metricsResponse = await this.callAPI('/api/cluster/metrics');
                console.log('=== CLUSTER METRICS API RESPONSE ===');
                console.log('Full response:', metricsResponse);
                console.log('Response data:', metricsResponse.data);
                console.log('Response success:', metricsResponse.success);
                if (metricsResponse.success && metricsResponse.data) {
                    this.clusterMetrics = metricsResponse.data;
                    console.log('Assigned clusterMetrics:', this.clusterMetrics);
                    this.updateClusterTableOnly(); // Update only the metrics columns
                } else {
                    console.log('No valid metrics data in response');
                }
            } catch (error) {
                console.error('Error fetching cluster metrics:', error);
            }
        }, 30000); // 30 seconds


        // SSH status updates - Update every hour
        setInterval(() => {
            console.log('Hourly SSH status check triggered');
            this.updateDashboardDisplay();
        }, 60 * 60 * 1000); // 1 hour in milliseconds

        // Load logs initially and set up polling for live logs
        this.loadLogs();
        setInterval(() => {
            this.loadLogs();
        }, 5000); // Update logs every 5 seconds
    }

    async loadNodes() {
        try {
            console.log('Loading nodes...');
            // Load nodes from the integrated API
            const response = await this.callAPI('/api/dashboard');
            console.log('Dashboard API response:', response);
            if (response.success && response.data && response.data.nodeData && Object.keys(response.data.nodeData).length > 0) {

                this.nodeData = response.data.nodeData;
                console.log('Loaded nodeData from dashboard API:', this.nodeData);
                this.updateDashboardDisplay();
                this.updateNodeStatusIndicators();
                this.populateNodeFilters();
            } else {
                console.log('Dashboard API did not return nodeData, trying nodes API...');
                // Fallback: load from nodes API if dashboard doesn't have nodeData
                const nodesResponse = await this.callAPI('/api/nodes');
                console.log('Nodes API response:', nodesResponse);
                if (nodesResponse.success && nodesResponse.data) {
                    // Convert API data to nodeData format
                    this.nodeData = {};
                    nodesResponse.data.forEach(node => {
                        const nodeId = node.name;
                        this.nodeData[nodeId] = {
                            cpu: 0,           // Will be updated with real data
                            memory: 0,        // Will be updated with real data
                            totalCpu: 4.0,
                            totalMemory: 8.0,
                            status: node.enabled ? 'active' : 'inactive',
                            host: node.host   // Store the host IP for matching with metrics target
                        };
                        console.log(`Converted node ${nodeId} with host ${node.host} and status ${this.nodeData[nodeId].status}`);
                    });
                    console.log('Final nodeData after conversion:', this.nodeData);

                    this.updateDashboardDisplay();
                    this.updateNodeStatusIndicators();
                    this.populateNodeFilters();
                }
            }
        } catch (error) {
            console.error('Error loading nodes:', error);
        }
    }

    async loadLogs() {
        try {
            const response = await this.callAPI('/api/logs?limit=50');
            if (response.success && response.data && response.data.logs) {
                this.logEntries = response.data.logs.map(log => ({
                    time: log.timestamp,
                    node: log.node,
                    module: log.module,
                    message: log.message,
                    type: log.type
                }));
                this.displayLogs(this.logEntries);
            }
        } catch (error) {
            console.error('Error loading logs:', error);
        }
    }

    populateNodeFilters() {
        // Populate log filter dropdown with actual node names
        const nodeFilter = this.elements.logNodeFilter;
        if (!nodeFilter) return;

        // Clear existing options except "All Nodes"
        while (nodeFilter.children.length > 1) {
            nodeFilter.removeChild(nodeFilter.lastChild);
        }

        // Add actual node names (include both active and error status nodes)
        Object.keys(this.nodeData).forEach(nodeId => {
            const option = document.createElement('option');
            option.value = nodeId;
            option.textContent = nodeId;
            nodeFilter.appendChild(option);
        });
    }




    async refreshRealData() {
        // Show loading state
        this.elements.syncBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Refreshing...</span>';
        this.elements.syncBtn.disabled = true;

        try {
            // Refresh real data from nodes
            await this.loadNodes();
            this.showNotification('Real node data refreshed successfully', 'success');
        } catch (error) {
            console.error('Error refreshing real data:', error);
            this.showNotification('Failed to refresh real data: ' + error.message, 'error');
        } finally {
            // Reset button state
            this.elements.syncBtn.innerHTML = '<span class="material-symbols-outlined transition-transform group-hover:rotate-180">sync</span><span>Refresh Real Data</span>';
            this.elements.syncBtn.disabled = false;
        }
    }



    updateMetrics() {
        // Only update display - no simulation
        // Real metrics are collected by the backend via SSH
        // Note: updateDashboardDisplay() is now called hourly for SSH status
    }

    async updateDashboardDisplay() {
        // Update cluster table dynamically
        const tbody = document.getElementById('cluster-table-body');
        tbody.innerHTML = ''; // Clear existing rows

        // Fetch SSH status for enabled nodes (only once per hour)
        const now = Date.now();
        const oneHour = 60 * 60 * 1000; // 1 hour in milliseconds

        if (now - this.lastSshStatusUpdate > oneHour || Object.keys(this.sshStatuses).length === 0) {
            try {
                console.log('Fetching SSH status (hourly update)...');
                const sshResponse = await this.callAPI('/api/ssh/status');
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

        Object.keys(this.nodeData).forEach(nodeId => {
            const node = this.nodeData[nodeId];
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            // Check if node is disabled (not enabled in the configuration)
            const isNodeDisabled = node.status === 'inactive';
            if (isNodeDisabled) {
                row.classList.add('opacity-60');
            }

            // Calculate memory usage in GB
            const usedMemory = node.totalMemory * (node.memory / 100);

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
                <td class="p-4 text-right number-animate" data-field="cpu">${(node.totalCpu - (node.totalCpu * node.cpu / 100)).toFixed(1)} / ${node.totalCpu} cores</td>
                <td class="p-4 text-right number-animate" data-field="memory">${usedMemory.toFixed(1)} / ${node.totalMemory} GB</td>
            `;

            tbody.appendChild(row);
        });

        // Calculate real-time CPU/Memory data for active and error nodes (nodes that are enabled but may have connection issues)
        // Calculate real-time CPU/Memory data for active and error nodes
        const displayableNodes = Object.values(this.nodeData).filter(node => node.status === 'active' || node.status === 'error');
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

                this.elements.cpuMemoryValue.textContent = `${availableCpu.toFixed(1)}/${avgRealCpu.toFixed(1)} cores / ${avgUsedMemory.toFixed(1)}/${avgRealMemory.toFixed(1)} GB`;
            } else {
                // Fallback to old calculation
                const totalAvgCpu = displayableNodes.reduce((sum, node) => sum + node.totalCpu, 0) / displayableNodes.length;
                const totalAvgMemory = displayableNodes.reduce((sum, node) => sum + node.totalMemory, 0) / displayableNodes.length;
                const avgCpuUsage = displayableNodes.reduce((sum, node) => sum + node.cpu, 0) / displayableNodes.length;
                const avgMemoryUsage = displayableNodes.reduce((sum, node) => sum + node.memory, 0) / displayableNodes.length;
                const availableCpu = totalAvgCpu - (totalAvgCpu * avgCpuUsage / 100);
                const usedMemory = totalAvgMemory * (avgMemoryUsage / 100);

                this.elements.cpuMemoryValue.textContent = `${availableCpu.toFixed(1)}/${totalAvgCpu.toFixed(1)} cores / ${usedMemory.toFixed(1)}/${totalAvgMemory.toFixed(1)} GB`;
            }
        }





    }

    animateNumber(element, newValue) {
        if (!element) return;

        const currentValue = parseInt(element.textContent.replace(/[^0-9]/g, '')) || 0;
        const difference = newValue - currentValue;

        if (Math.abs(difference) > 50) { // Only animate significant changes
            element.style.transform = 'scale(1.05)';
            element.textContent = Math.round(newValue).toLocaleString();

            setTimeout(() => {
                element.style.transform = 'scale(1)';
            }, 200);
        }
    }

    updateTrend(element, direction) {
        if (!element) return;

        const isPositive = direction === 'up';
        const value = (Math.random() * 2).toFixed(1);
        const symbol = isPositive ? 'trending_up' : 'trending_down';
        const colorClass = isPositive ? 'text-success' : 'text-danger';

        element.textContent = `${isPositive ? '+' : '-'}${value}%`;
        element.className = `flex items-center gap-1 text-sm ${colorClass} dark:${colorClass}-dark mt-1`;

        const icon = element.querySelector('.material-symbols-outlined');
        if (icon) icon.textContent = symbol;
    }


    filterLogs() {
        const nodeFilter = this.elements.logNodeFilter.value;
        const moduleFilter = this.elements.logModuleFilter.value;

        const filteredLogs = this.logEntries.filter(log => {
            const nodeMatch = nodeFilter === 'All Nodes' || log.node === nodeFilter;
            const moduleMatch = moduleFilter === 'All Modules' || log.module === moduleFilter;
            return nodeMatch && moduleMatch;
        });

        this.displayLogs(filteredLogs);
    }

    displayLogs(logs) {
        const container = this.elements.logsContainer;
        container.innerHTML = '';

        logs.forEach((log, index) => {
            const logElement = document.createElement('p');
            logElement.className = 'animate-fade-in';
            logElement.style.animationDelay = `${index * 50}ms`;

            const typeClass = this.getLogTypeClass(log.type);
            logElement.innerHTML = `
                <span class="text-sky-400">${log.time}</span> - 
                <span class="text-purple-400">${log.node}</span> - 
                <span class="${typeClass}">${log.module}</span>: ${log.message}
            `;

            container.appendChild(logElement);
        });
    }

    getLogTypeClass(type) {
        const classes = {
            info: 'text-emerald-400',
            warning: 'text-yellow-400',
            error: 'text-red-400',
            success: 'text-emerald-400',
            metric: 'text-emerald-400'
        };
        return classes[type] || 'text-gray-400';
    }

    addRandomLog() {
        const randomMessages = [
            'Processing batch request...',
            'Cache updated successfully',
            'Connection pool optimized',
            'Load balancer adjusted',
            'Memory usage optimized',
            'Network latency stable',
            'Database query optimized'
        ];

        const randomMessage = randomMessages[Math.floor(Math.random() * randomMessages.length)];
        const currentTime = new Date().toISOString().slice(0, 19).replace('T', ' ');
        const activeNodes = Object.keys(this.nodeData).filter(id => this.nodeData[id].status === 'active' || this.nodeData[id].status === 'error');
        const randomNode = activeNodes[Math.floor(Math.random() * activeNodes.length)];

        const newLog = {
            time: currentTime,
            node: randomNode.charAt(0).toUpperCase() + randomNode.slice(1),
            module: 'Module A',
            message: randomMessage,
            type: 'info'
        };

        this.logEntries.unshift(newLog);
        if (this.logEntries.length > 50) {
            this.logEntries.pop(); // Keep only latest 50 logs
        }

        this.filterLogs(); // Refresh the display
    }

    startRealTimeUpdates() {
        // Initial display - fetch SSH status immediately on startup
        //this.updateDashboardDisplay();
        this.updateNodeStatusIndicators();
        this.refreshNodesTable();
        this.loadLogs(); // Load real logs instead of static ones
        this.displayLogs(this.logEntries);

        // Set up WebSocket connection for real-time updates
        this.setupWebSocket();
    }

    setupWebSocket() {
        // In a real implementation, this would connect to the backend WebSocket
        // For now, we'll simulate real-time updates with intervals
        console.log('WebSocket connection would be established here');
    }

    async callAPI(endpoint, method = 'GET', data = null) {
        try {
            const config = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                },
            };

            if (data) {
                config.body = JSON.stringify(data);
            }

            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, config);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            return await response.json();
        } catch (error) {
            console.error('API call failed:', error);
            // Return mock response for demonstration
            return {
                success: true,
                message: 'Mock API response',
                data: {}
            };
        }
    }

    async refreshDashboard() {
        try {
            // Reload nodes from API
            await this.loadNodes();
            // Also refresh the nodes table if modal is open
            if (this.elements.nodeManagementModal && !this.elements.nodeManagementModal.classList.contains('hidden')) {
                this.refreshNodesTable();
            }
        } catch (error) {
            console.error('Error refreshing dashboard:', error);
        }
    }

    updateNodeStatusIndicators() {
        // Update node status indicators based on real node data
        const nodeIds = Object.keys(this.nodeData);
        const container = this.elements.nodeStatusIndicatorsContainer;
        container.innerHTML = ''; // Clear existing indicators

        nodeIds.forEach((nodeId, index) => {
            const node = this.nodeData[nodeId];
            const statusElement = document.createElement('span');
            statusElement.className = `h-4 w-4 rounded-full ${node.status === 'active' ? 'bg-success animate-node-pulse' : 'bg-danger'}`;
            statusElement.style.animationDelay = `${index * 0.2}s`;
            statusElement.title = `${nodeId}: ${node.status === 'active' ? 'Active' : 'Inactive'}`;

            container.appendChild(statusElement);
        });
    }

    // Legacy function for backward compatibility (browser cache)
    toggleNodeManagement() {
        console.log('toggleNodeManagement called - redirecting to modal');
        console.log('Modal element available:', !!this.elements.nodeManagementModal);
        console.log('Modal element ID:', this.elements.nodeManagementModal?.id);
        this.openNodeManagementModal();
    }

    openNodeManagementModal() {
        console.log('Opening node management modal');
        console.log('Available elements:', Object.keys(this.elements));
        console.log('Modal element search:', document.getElementById('node-management-modal'));
        const modal = this.elements.nodeManagementModal;

        if (!modal) {
            console.error('Node management modal not found!');
            console.error('Current modal element:', modal);
            console.error('Direct DOM query:', document.getElementById('node-management-modal'));
            return;
        }

        modal.classList.remove('hidden');
        // Refresh nodes table when modal opens
        this.refreshNodesTable();
    }

    closeNodeManagementModal() {
        console.log('Closing node management modal');
        const modal = this.elements.nodeManagementModal;

        if (!modal) {
            console.error('Node management modal not found!');
            return;
        }

        modal.classList.add('hidden');
        // Clear form when closing modal
        this.clearNodeForm();
    }

    async addNode() {
        // If in edit mode, call update instead
        if (this.isEditMode) {
            await this.updateNode();
            return;
        }

        const nodeData = {
            host: this.elements.nodeHost.value,
            user: this.elements.nodeUser.value,
            key_path: this.elements.nodeKeypath.value,
            conf_dir: this.elements.nodeConfdir.value,
            binary_dir: this.elements.nodeBindir.value,
            description: this.elements.nodeDescription.value,
            enabled: this.elements.nodeEnabled.checked
        };

        if (!nodeData.host || !nodeData.user || !nodeData.key_path || !nodeData.conf_dir || !nodeData.binary_dir) {
            this.showNotification('Please fill in all required fields', 'error');
            return;
        }

        this.setButtonLoading(this.elements.addNodeBtn, true);

        try {
            const nodeName = this.elements.nodeName.value || `node-${Date.now()}`;
            const response = await this.callAPI(`/api/nodes/${nodeName}`, 'POST', nodeData);

            if (response.success) {
                this.showNotification(`Node ${nodeName} added successfully`, 'success');
                this.clearNodeForm();
                this.refreshNodesTable();
                this.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to add node');
            }
        } catch (error) {
            console.error('Error adding node:', error);
            this.showNotification('Failed to add node: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.elements.addNodeBtn, false);
        }
    }

    clearNodeForm() {
        this.elements.nodeName.value = '';
        this.elements.nodeHost.value = '';
        this.elements.nodeUser.value = '';
        this.elements.nodeKeypath.value = '';
        this.elements.nodeConfdir.value = '';
        this.elements.nodeBindir.value = '';
        this.elements.nodeDescription.value = '';
        this.elements.nodeEnabled.checked = true;

        // Exit edit mode if active
        if (this.isEditMode) {
            this.exitEditMode();
        }
    }

    async refreshNodesTable() {
        try {
            const response = await this.callAPI('/api/nodes');
            if (response.success && response.data) {
                this.displayNodesTable(response.data);
            }
        } catch (error) {
            console.error('Error refreshing nodes table:', error);
        }
    }

    displayNodesTable(nodes) {
        const tbody = this.elements.nodesTableBody;
        tbody.innerHTML = '';

        if (nodes.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="5" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No nodes configured</td>';
            tbody.appendChild(row);
            return;
        }

        nodes.forEach(node => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const statusBadge = node.enabled
                ? '<div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark"><span class="h-2 w-2 rounded-full bg-success"></span>Enabled</div>'
                : '<div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark"><span class="h-2 w-2 rounded-full bg-danger"></span>Disabled</div>';

            row.innerHTML = `
                <td class="p-4 font-medium">${node.name}</td>
                <td class="p-4">${node.host}</td>
                <td class="p-4">${statusBadge}</td>
                <td class="p-4">${node.description || '-'}</td>
                <td class="p-4">
                    <div class="flex items-center gap-2">
                        <button onclick="window.vuDataSimManager.editNode('${node.name}')" class="px-3 py-1 text-xs rounded bg-primary/20 text-primary hover:bg-primary/30 transition-colors">
                            <span class="material-symbols-outlined text-sm mr-1">edit</span>
                            Edit
                        </button>
                        <button onclick="window.vuDataSimManager.toggleNode('${node.name}', ${!node.enabled})" class="px-3 py-1 text-xs rounded ${node.enabled ? 'bg-danger/20 text-danger hover:bg-danger/30' : 'bg-success/20 text-success hover:bg-success/30'} transition-colors">
                            ${node.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button onclick="window.vuDataSimManager.removeNode('${node.name}')" class="px-3 py-1 text-xs rounded bg-danger/20 text-danger hover:bg-danger/30 transition-colors">
                            Remove
                        </button>
                    </div>
                </td>
            `;

            tbody.appendChild(row);
        });
    }

    async toggleNode(nodeName, enable) {
        try {
            const response = await this.callAPI(`/api/nodes/${nodeName}`, 'PUT', { enabled: enable });

            if (response.success) {
                this.showNotification(`Node ${nodeName} ${enable ? 'enabled' : 'disabled'} successfully`, 'success');
                this.refreshNodesTable();
                this.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to update node');
            }
        } catch (error) {
            console.error('Error toggling node:', error);
            this.showNotification('Failed to update node: ' + error.message, 'error');
        }
    }

    async removeNode(nodeName) {
        if (!confirm(`Are you sure you want to remove node "${nodeName}"?`)) {
            return;
        }

        try {
            const response = await this.callAPI(`/api/nodes/${nodeName}`, 'DELETE');

            if (response.success) {
                this.showNotification(`Node ${nodeName} removed successfully`, 'success');
                this.refreshNodesTable();
                this.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to remove node');
            }
        } catch (error) {
            console.error('Error removing node:', error);
            this.showNotification('Failed to remove node: ' + error.message, 'error');
        }
    }

    editNode(nodeName) {
        console.log('Editing node:', nodeName);
        // For now, we'll use the same form but in edit mode
        // In a real implementation, you might want to pre-populate the form with existing data
        this.isEditMode = true;
        this.editNodeName = nodeName;

        // Update the form title and button text
        const formTitle = document.querySelector('#node-management-modal h4');
        const addButton = this.elements.addNodeBtn;

        if (formTitle) formTitle.textContent = `Edit Node: ${nodeName}`;
        if (addButton) {
            addButton.innerHTML = '<span class="material-symbols-outlined">save</span><span>Update Node</span>';
            addButton.onclick = () => this.updateNode();
        }

        this.showNotification(`Edit mode for node ${nodeName}`, 'info');
    }

    async updateNode() {
        if (!this.editNodeName) {
            this.showNotification('No node selected for editing', 'error');
            return;
        }

        const nodeData = {
            host: this.elements.nodeHost.value,
            user: this.elements.nodeUser.value,
            key_path: this.elements.nodeKeypath.value,
            conf_dir: this.elements.nodeConfdir.value,
            binary_dir: this.elements.nodeBindir.value,
            description: this.elements.nodeDescription.value,
            enabled: this.elements.nodeEnabled.checked
        };

        if (!nodeData.host || !nodeData.user || !nodeData.key_path || !nodeData.conf_dir || !nodeData.binary_dir) {
            this.showNotification('Please fill in all required fields', 'error');
            return;
        }

        this.setButtonLoading(this.elements.addNodeBtn, true);

        try {
            // For now, we'll use PUT method to update the node
            // Note: This would need backend support for updating nodes
            const response = await this.callAPI(`/api/nodes/${this.editNodeName}`, 'PUT', nodeData);

            if (response.success) {
                this.showNotification(`Node ${this.editNodeName} updated successfully`, 'success');
                this.clearNodeForm();
                this.refreshNodesTable();
                this.loadNodes(); // Refresh the dashboard nodes too
                this.exitEditMode();
            } else {
                throw new Error(response.message || 'Failed to update node');
            }
        } catch (error) {
            console.error('Error updating node:', error);
            this.showNotification('Failed to update node: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.elements.addNodeBtn, false);
        }
    }

    exitEditMode() {
        this.isEditMode = false;
        this.editNodeName = null;

        // Reset form title and button
        const formTitle = document.querySelector('#node-management-modal h4');
        const addButton = this.elements.addNodeBtn;

        if (formTitle) formTitle.textContent = 'Add New Node';
        if (addButton) {
            addButton.innerHTML = '<span class="material-symbols-outlined">add</span><span>Add Node</span>';
            addButton.onclick = () => this.addNode();
        }
    }

    setButtonLoading(button, loading) {
        if (!button) return;

        if (loading) {
            button.disabled = true;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Processing...</span>';
        } else {
            button.disabled = false;
            if (this.isEditMode) {
                button.innerHTML = '<span class="material-symbols-outlined">save</span><span>Update Node</span>';
            } else {
                button.innerHTML = '<span class="material-symbols-outlined">add</span><span>Add Node</span>';
            }
        }
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 animate-slide-down ${type === 'success' ? 'bg-success text-white' :
                type === 'error' ? 'bg-danger text-white' :
                    type === 'warning' ? 'bg-yellow-500 text-white' :
                        'bg-primary text-white'
            }`;
        notification.textContent = message;

        document.body.appendChild(notification);

        // Remove after 3 seconds
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }

    // Binary Control Methods

    openBinaryControlModal() {
        console.log('Opening binary control modal');
        const modal = this.elements.binaryControlModal;

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
        const modal = this.elements.binaryControlModal;

        if (!modal) {
            console.error('Binary control modal not found!');
            return;
        }

        modal.classList.add('hidden');
        this.clearBinaryForm();
    }

    populateBinaryNodeSelect() {
        const nodeSelect = this.elements.binaryNodeSelect;
        if (!nodeSelect) return;

        console.log('Populating binary node select with nodeData:', this.nodeData);

        // Clear existing options except the first one
        while (nodeSelect.children.length > 1) {
            nodeSelect.removeChild(nodeSelect.lastChild);
        }

        // Add enabled nodes (both active and error status nodes are enabled)
        let addedNodes = 0;
        Object.keys(this.nodeData).forEach(nodeId => {
            const node = this.nodeData[nodeId];
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
            const response = await this.callAPI('/api/binary/status');
            if (response.success && response.data) {
                this.displayAllBinaryStatuses(response.data);
            }
        } catch (error) {
            console.error('Error refreshing all binary statuses:', error);
        }
    }

    displayAllBinaryStatuses(statuses) {
        const tbody = this.elements.binaryAllStatusBody;
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
        const selectedNode = this.elements.binaryNodeSelect.value;
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
            const nodesResponse = await this.callAPI('/api/nodes');
            if (nodesResponse.success && nodesResponse.data) {
                const nodeData = nodesResponse.data.find(node => node.name === selectedNode);
                if (nodeData) {
                    const binaryPath = `${nodeData.binary_dir}/finalvudatasim`;
                    this.elements.binaryPathDisplay.textContent = binaryPath;
                    this.elements.binaryNameDisplay.textContent = 'finalvudatasim';
                    this.elements.binaryPathInfo.classList.remove('hidden');

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
        this.elements.binaryPathInfo.classList.add('hidden');
    }

    async refreshBinaryStatus() {
        const selectedNode = this.elements.binaryNodeSelect.value;
        if (!selectedNode) {
            this.showNotification('Please select a node first', 'warning');
            return;
        }

        try {
            const response = await this.callAPI(`/api/binary/status/${selectedNode}`);
            if (response.success && response.data) {
                this.displayBinaryStatus(response.data);
            }
        } catch (error) {
            console.error('Error refreshing binary status:', error);
            this.showNotification('Failed to refresh binary status: ' + error.message, 'error');
        }
    }

    displayBinaryStatus(status) {
        this.elements.binaryStatusDisplay.classList.remove('hidden');
        this.elements.binaryCurrentStatus.textContent = status.status;
        this.elements.binaryCurrentPid.textContent = status.pid || '-';
        this.elements.binaryCurrentStartTime.textContent = status.startTime || '-';
        this.elements.binaryCurrentLastChecked.textContent = status.lastChecked || '-';
        this.elements.binaryCurrentProcessInfo.textContent = status.processInfo || '-';

        // Update button states based on status
        this.updateBinaryControlButtons(status.status);

        // Show additional binary path information if available
        const selectedNode = this.elements.binaryNodeSelect.value;
        if (selectedNode) {
            this.addSshOutput(`=== Status Check for Node: ${selectedNode} ===`);
            this.addSshOutput(`Status: ${status.status} | PID: ${status.pid || 'N/A'} | Last Checked: ${status.lastChecked || 'N/A'}`);
        }
    }

    updateBinaryControlButtons(status) {
        const startBtn = this.elements.startBinaryBtn;
        const stopBtn = this.elements.stopBinaryBtn;

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
        this.elements.binaryStatusDisplay.classList.add('hidden');
        this.elements.startBinaryBtn.disabled = true;
        this.elements.stopBinaryBtn.disabled = true;
    }

    async startBinary() {
        const selectedNode = this.elements.binaryNodeSelect.value;
        const timeout = parseInt(this.elements.binaryTimeoutInput.value) || 30;

        if (!selectedNode) {
            this.showNotification('Please select a node first', 'warning');
            return;
        }

        this.setBinaryButtonLoading('start', true);

        try {
            this.addSshOutput(`=== Starting Binary on Node: ${selectedNode} ===`);
            this.addSshOutput(`Checking binary existence...`);

            const response = await this.callAPI(`/api/binary/start/${selectedNode}?timeout=${timeout}`, 'POST');

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
                this.showNotification(`Binary start initiated on node ${selectedNode}`, 'success');

                // Refresh status after a delay
                setTimeout(() => {
                    this.refreshBinaryStatus();
                    this.refreshAllBinaryStatuses();
                }, 3000);
            } else {
                // Handle specific error cases
                if (response.data && response.data.error === 'BINARY_NOT_FOUND') {
                    this.addSshOutput(`✗ ERROR: Binary not found at ${response.data.binaryPath}`);
                    this.showNotification(`Binary not found on node ${selectedNode}. Please check deployment.`, 'error');
                } else {
                    throw new Error(response.message || 'Failed to start binary');
                }
            }
        } catch (error) {
            console.error('Error starting binary:', error);
            this.showNotification('Failed to start binary: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        } finally {
            this.setBinaryButtonLoading('start', false);
        }
    }

    async stopBinary() {
        const selectedNode = this.elements.binaryNodeSelect.value;
        const timeout = parseInt(this.elements.binaryTimeoutInput.value) || 30;

        if (!selectedNode) {
            this.showNotification('Please select a node first', 'warning');
            return;
        }

        this.setBinaryButtonLoading('stop', true);

        try {
            this.addSshOutput(`=== Stopping Binary on Node: ${selectedNode} ===`);
            this.addSshOutput(`Timeout: ${timeout} seconds`);
            this.addSshOutput(`Executing stop command...`);

            const response = await this.callAPI(`/api/binary/stop/${selectedNode}?timeout=${timeout}`, 'POST');
            if (response.success) {
                this.addSshOutput(`✓ Binary stop command sent successfully`);
                if (response.data && response.data.previousPID) {
                    this.addSshOutput(`✓ Previous PID: ${response.data.previousPID}`);
                }
                this.showNotification(`Binary stop initiated on node ${selectedNode}`, 'success');

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
            this.showNotification('Failed to stop binary: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        } finally {
            this.setBinaryButtonLoading('stop', false);
        }
    }

    // Add method to get current binary output
    async getBinaryOutput() {
        const selectedNode = this.elements.binaryNodeSelect.value;
        if (!selectedNode) {
            this.showNotification('Please select a node first', 'warning');
            return;
        }

        try {
            this.addSshOutput(`=== Getting Current Output for Node: ${selectedNode} ===`);

            // Get node configuration to find log file path
            const nodesResponse = await this.callAPI('/api/nodes');
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
                    this.showNotification('Output retrieval requires backend API implementation', 'info');
                }
            }
        } catch (error) {
            console.error('Error getting binary output:', error);
            this.showNotification('Failed to get binary output: ' + error.message, 'error');
            this.addSshOutput(`✗ ERROR: ${error.message}`);
        }
    }

    setBinaryButtonLoading(action, loading) {
        const startBtn = this.elements.startBinaryBtn;
        const stopBtn = this.elements.stopBinaryBtn;

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
        const outputContainer = this.elements.binarySshOutput;
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
        this.elements.binaryNodeSelect.value = '';
        this.hideBinaryPathInfo();
        this.hideBinaryStatusDisplay();
        this.elements.binaryTimeoutInput.value = '30';
        this.elements.binarySshOutput.innerHTML = '<div class="text-text-secondary-light dark:text-text-secondary-dark">SSH output will appear here...</div>';
    }

    // ClickHouse Metrics Methods

    openClickHouseMetricsModal() {
        console.log('Opening ClickHouse metrics modal');
        const modal = this.elements.clickHouseMetricsModal;

        if (!modal) {
            console.error('ClickHouse metrics modal not found!');
            return;
        }

        modal.classList.remove('hidden');
        this.refreshClickHouseMetrics();
    }

    closeClickHouseMetricsModal() {
        console.log('Closing ClickHouse metrics modal');
        const modal = this.elements.clickHouseMetricsModal;

        if (!modal) {
            console.error('ClickHouse metrics modal not found!');
            return;
        }

        modal.classList.add('hidden');
    }

    async refreshClickHouseMetrics() {
        const refreshBtn = this.elements.refreshClickHouseMetricsBtn;
        if (refreshBtn) {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Refreshing...</span>';
        }

        try {
            // First check ClickHouse health
            const healthResponse = await this.callAPI('/api/clickhouse/health');
            if (healthResponse.success && healthResponse.data) {
                this.updateClickHouseStatus(healthResponse.data);
            }

            // Then get metrics
            const metricsResponse = await this.callAPI('/api/clickhouse/metrics');
            console.log('ClickHouse metrics API response:', metricsResponse);
            if (metricsResponse.success && metricsResponse.data) {
                console.log('ClickHouse metrics data:', metricsResponse.data);
                this.displayClickHouseMetrics(metricsResponse.data);
                this.showNotification('ClickHouse metrics refreshed successfully', 'success');
            } else {
                this.showNotification('Failed to load ClickHouse metrics', 'error');
            }
        } catch (error) {
            console.error('Error refreshing ClickHouse metrics:', error);
            this.showNotification('Failed to refresh ClickHouse metrics: ' + error.message, 'error');
            this.updateClickHouseStatus({ status: 'error', error: error.message });
        } finally {
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<span class="material-symbols-outlined">refresh</span><span>Refresh Metrics</span>';
            }
        }
    }

    updateClickHouseStatus(healthData) {
        const statusElement = this.elements.clickHouseStatus;
        const lastUpdateElement = this.elements.clickHouseLastUpdate;

        if (statusElement) {
            let statusBadge = '';
            if (healthData.status === 'connected') {
                statusBadge = '<div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark"><span class="h-2 w-2 rounded-full bg-success"></span>Connected</div>';
            } else if (healthData.status === 'error') {
                statusBadge = '<div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark"><span class="h-2 w-2 rounded-full bg-danger"></span>Error</div>';
            } else {
                statusBadge = '<div class="inline-flex items-center gap-2 rounded-full bg-warning/20 dark:bg-warning-dark/20 px-3 py-1 text-xs font-medium text-warning dark:text-warning-dark"><span class="h-2 w-2 rounded-full bg-warning"></span>Disconnected</div>';
            }
            statusElement.innerHTML = statusBadge;
        }

        if (lastUpdateElement) {
            const timestamp = healthData.last_checked ? new Date(healthData.last_checked).toLocaleString() : 'Never';
            lastUpdateElement.textContent = `Last checked: ${timestamp}`;
        }
    }

    displayClickHouseMetrics(metrics) {
        console.log('Received ClickHouse metrics:', metrics);

        // Display Pod Metrics first
        this.displayPodResourceMetrics(metrics.podResourceMetrics || []);
        this.displayPodStatusMetrics(metrics.podStatusMetrics || []);

        // Display System Metrics
        this.displaySystemMetrics(metrics.systemMetrics || []);

        // Display Database Metrics
        this.displayDatabaseMetrics(metrics.databaseMetrics || []);

        // Display Container Metrics
        this.displayContainerMetrics(metrics.containerMetrics || []);

        // Update last update time
        if (this.elements.clickHouseLastUpdate && metrics.lastUpdated) {
            const timestamp = new Date(metrics.lastUpdated).toLocaleString();
            this.elements.clickHouseLastUpdate.textContent = `Last updated: ${timestamp}`;
        }
    }

    displayPodResourceMetrics(metrics) {
        console.log('Displaying pod resource metrics:', metrics);
        const tbody = this.elements.podResourceMetricsTable;
        if (!tbody) {
            console.error('Pod resource metrics table not found in elements');
            return;
        }

        tbody.innerHTML = '';

        if (!metrics || !Array.isArray(metrics) || metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="5" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No pod resource metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            try {
                const row = document.createElement('tr');
                row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

                // Handle potential missing or invalid values
                const clusterId = metric.clusterId || 'N/A';
                const podName = metric.podName || 'N/A';
                const cpuPercentage = typeof metric.cpuPercentage === 'number' ? metric.cpuPercentage.toFixed(2) : 'N/A';
                const memoryPercentage = typeof metric.memoryPercentage === 'number' ? metric.memoryPercentage.toFixed(2) : 'N/A';
                let timestamp = 'N/A';
                try {
                    if (metric.lastTimestamp) {
                        timestamp = new Date(metric.lastTimestamp).toLocaleString();
                    }
                } catch (error) {
                    console.warn('Invalid timestamp format:', error);
                }

                row.innerHTML = `
                    <td class="p-3">${clusterId}</td>
                    <td class="p-3">${podName}</td>
                    <td class="p-3 text-right">${cpuPercentage}${typeof metric.cpuPercentage === 'number' ? '%' : ''}</td>
                    <td class="p-3 text-right">${memoryPercentage}${typeof metric.memoryPercentage === 'number' ? '%' : ''}</td>
                    <td class="p-3">${timestamp}</td>
                `;
                tbody.appendChild(row);
            } catch (error) {
                console.error('Error processing pod resource metric:', error);
            }
        });
    }

    displayPodStatusMetrics(metrics) {
        console.log('Displaying pod status metrics:', metrics);
        const tbody = this.elements.podStatusMetricsTable;
        if (!tbody) {
            console.error('Pod status metrics table not found in elements');
            return;
        }

        tbody.innerHTML = '';

        if (!metrics || !Array.isArray(metrics) || metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="7" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No pod status metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';
            const statusClass = String(metric.podPhase).toLowerCase() === 'running' ? 'text-success' : 'text-danger';

            // Handle any undefined values with default strings
            const clusterId = metric.clusterId || 'N/A';
            const nodeName = metric.nodeName || 'N/A';
            const podName = metric.podName || 'N/A';
            const podPhase = metric.podPhase || 'Unknown';
            const containerStatus = metric.containerStatus || 'Unknown';
            const runningContainers = typeof metric.runningContainers === 'number' ? metric.runningContainers : 0;
            const nonRunningContainers = typeof metric.nonRunningContainers === 'number' ? metric.nonRunningContainers : 0;
            const derivedStatus = metric.derivedStatus || 'Unknown';

            row.innerHTML = `
                <td class="p-3">${clusterId}</td>
                <td class="p-3">${nodeName}</td>
                <td class="p-3">${podName}</td>
                <td class="p-3">${podPhase}</td>
                <td class="p-3">${containerStatus}</td>
                <td class="p-3">${runningContainers}/${runningContainers + nonRunningContainers}</td>
                <td class="p-3 ${statusClass}">${derivedStatus}</td>
            `;
            tbody.appendChild(row);
        });
    }

    displayKafkaMetrics(metrics) {
        const tbody = this.elements.kafkaMetricsTable?.querySelector('tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="7" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No Kafka producer metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const timestamp = new Date(metric.timestamp).toLocaleString();
            const recordRate = metric.recordSendRate ? metric.recordSendRate.toFixed(2) : '0.00';
            const byteRate = metric.byteRate ? (metric.byteRate / 1024 / 1024).toFixed(2) : '0.00'; // Convert to MB/s
            const compressionRate = metric.compressionRate ? metric.compressionRate.toFixed(2) : '0.00';

            row.innerHTML = `
                <td class="p-3 font-medium">${timestamp}</td>
                <td class="p-3">${metric.clientId || '-'}</td>
                <td class="p-3">${metric.topic || '-'}</td>
                <td class="p-3 text-right">${metric.recordSendTotal || 0}</td>
                <td class="p-3 text-right">${recordRate}/s</td>
                <td class="p-3 text-right">${byteRate} MB/s</td>
                <td class="p-3 text-right">${compressionRate}%</td>
            `;

            tbody.appendChild(row);
        });
    }

    displaySystemMetrics(metrics) {
        const tbody = this.elements.systemMetricsTable?.querySelector('tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="6" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No system metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const timestamp = new Date(metric.timestamp).toLocaleString();
            const cpuUsage = metric.cpuUsage ? metric.cpuUsage.toFixed(1) : '0.0';
            const memoryUsage = metric.memoryUsage ? metric.memoryUsage.toFixed(1) : '0.0';
            const diskUsage = metric.diskUsage ? metric.diskUsage.toFixed(1) : '0.0';
            const networkRx = metric.networkRx ? (metric.networkRx / 1024 / 1024).toFixed(2) : '0.00'; // Convert to MB
            const networkTx = metric.networkTx ? (metric.networkTx / 1024 / 1024).toFixed(2) : '0.00'; // Convert to MB

            row.innerHTML = `
                <td class="p-3 font-medium">${timestamp}</td>
                <td class="p-3">${metric.host || '-'}</td>
                <td class="p-3 text-right">${cpuUsage}%</td>
                <td class="p-3 text-right">${memoryUsage}%</td>
                <td class="p-3 text-right">${diskUsage}%</td>
                <td class="p-3 text-right">${networkRx}/${networkTx} MB</td>
            `;

            tbody.appendChild(row);
        });
    }

    displayDatabaseMetrics(metrics) {
        const tbody = this.elements.databaseMetricsTable?.querySelector('tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="6" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No database metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const timestamp = new Date(metric.timestamp).toLocaleString();
            const queryDuration = metric.queryDuration ? metric.queryDuration.toFixed(2) : '0.00';

            row.innerHTML = `
                <td class="p-3 font-medium">${timestamp}</td>
                <td class="p-3">${metric.database || '-'}</td>
                <td class="p-3">${metric.table || '-'}</td>
                <td class="p-3 text-right">${metric.queryCount || 0}</td>
                <td class="p-3 text-right">${queryDuration}ms</td>
                <td class="p-3 text-right">${metric.errorCount || 0}</td>
            `;

            tbody.appendChild(row);
        });
    }

    displayContainerMetrics(metrics) {
        const tbody = this.elements.containerMetricsTable?.querySelector('tbody');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="6" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No container metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const timestamp = new Date(metric.timestamp).toLocaleString();
            const cpuUsage = metric.cpuUsage ? metric.cpuUsage.toFixed(1) : '0.0';
            const memoryUsage = metric.memoryUsage ? metric.memoryUsage.toFixed(1) : '0.0';

            row.innerHTML = `
                <td class="p-3 font-medium">${timestamp}</td>
                <td class="p-3">${metric.namespace || '-'}</td>
                <td class="p-3">${metric.podName || '-'}</td>
                <td class="p-3">${metric.containerName || '-'}</td>
                <td class="p-3 text-right">${cpuUsage}%</td>
                <td class="p-3 text-right">${memoryUsage}%</td>
                <td class="p-3">${metric.status || '-'}</td>
            `;

            tbody.appendChild(row);
        });
    }

    // O11y Source Management Methods

    loadO11ySources() {
        console.log('Loading o11y sources...');
        this.callAPI('/api/o11y/sources')
            .then(response => {
                console.log('O11y sources API response:', response);
                if (response.success && response.data) {
                    console.log('Sources data:', response.data);
                    this.populateO11ySourcesSelect(response.data);
                    console.log('Loaded o11y sources:', response.data.length, 'sources');
                } else {
                    console.error('Failed to load o11y sources:', response.message);
                    this.showNotification('Failed to load O11y sources: ' + response.message, 'error');
                }
            })
            .catch(error => {
                console.error('Error loading o11y sources:', error);
                this.showNotification('Error loading O11y sources: ' + error.message, 'error');
            });
    }

    populateO11ySourcesSelect(sources) {
        console.log('populateO11ySourcesSelect called with:', sources);
        const list = this.elements.o11ySourcesList;
        if (!list) {
            console.error('o11ySourcesList element not found!');
            return;
        }

        console.log('o11ySourcesList element found:', list);

        // Store sources for filtering
        this.o11ySources = sources;

        // Clear existing options
        list.innerHTML = '';

        if (!sources || sources.length === 0) {
            console.error('No sources provided or sources array is empty');
            list.innerHTML = '<div class="o11y-sources-empty"><span class="material-symbols-outlined">error</span><p>No O11y sources available</p></div>';
            return;
        }

        // Add sources as custom options with checkboxes
        sources.forEach((source, index) => {
            console.log(`Adding source ${index + 1}: ${source}`);
            const option = document.createElement('div');
            option.className = 'o11y-source-option';
            option.innerHTML = `
                <div class="o11y-source-checkbox" data-source="${source}"></div>
                <span class="o11y-source-label">${source}</span>
            `;

            option.addEventListener('click', () => {
                console.log('Source clicked:', source);
                this.toggleO11ySource(source);
            });
            list.appendChild(option);
        });

        // Initialize selected sources array
        this.selectedO11ySources = [];

        console.log(`Successfully populated ${sources.length} o11y sources`);
    }

    syncConfigs() {
        const selectedSources = [...this.selectedO11ySources];
        const selectedEPS = parseInt(this.elements.epsSelect.value);

        if (selectedSources.length === 0) {
            this.showNotification('Please select at least one o11y source', 'warning');
            return;
        }

        if (!selectedEPS || selectedEPS <= 0) {
            this.showNotification('Please select a valid EPS target', 'warning');
            return;
        }

        // Show loading state
        this.setSyncButtonLoading(true);

        console.log('Syncing configs for sources:', selectedSources, 'EPS:', selectedEPS);

        // Call EPS distribution API first
        this.callAPI('/api/o11y/eps/distribute', 'POST', {
            selectedSources: selectedSources,
            totalEps: selectedEPS
        })
        .then(epsResponse => {
            console.log('EPS distribution response:', epsResponse);
            if (!epsResponse.success) {
                throw new Error(epsResponse.message || 'EPS distribution failed');
            }

            // Call conf.d distribution API
            return this.callAPI('/api/o11y/confd/distribute', 'POST');
        })
        .then(confDResponse => {
            console.log('Conf.d distribution response:', confDResponse);
            if (!confDResponse.success) {
                throw new Error(confDResponse.message || 'Conf.d distribution failed');
            }

            // Both APIs succeeded
            this.showSyncSuccess();
            this.showNotification('Configs synced successfully!', 'success');
        })
        .catch(error => {
            console.error('Error syncing configs:', error);
            this.showSyncError(error.message);
            this.showNotification('Failed to sync configs: ' + error.message, 'error');
        })
        .finally(() => {
            this.setSyncButtonLoading(false);
        });
    }

    setSyncButtonLoading(loading) {
        const button = this.elements.syncConfigsBtn;
        if (!button) return;

        if (loading) {
            button.disabled = true;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Syncing...</span>';
        } else {
            button.disabled = false;
            button.innerHTML = '<span class="material-symbols-outlined">sync</span><span>Sync Configs</span>';
        }
    }

    showSyncSuccess() {
        this.hideSyncMessages();
        this.elements.syncSuccessMessage.classList.remove('hidden');
        this.elements.syncStatusContainer.classList.remove('hidden');

        // Auto-hide after 5 seconds
        setTimeout(() => {
            this.hideSyncMessages();
        }, 5000);
    }

    showSyncError(message) {
        this.hideSyncMessages();
        this.elements.syncErrorMessage.classList.remove('hidden');
        this.elements.syncStatusContainer.classList.remove('hidden');

        // Auto-hide after 8 seconds for errors
        setTimeout(() => {
            this.hideSyncMessages();
        }, 8000);
    }

    hideSyncMessages() {
        this.elements.syncSuccessMessage.classList.add('hidden');
        this.elements.syncErrorMessage.classList.add('hidden');
        this.elements.syncStatusContainer.classList.add('hidden');
    }

    // Custom Multi-Select Methods

    toggleO11ySourcesDropdown() {
        const isOpen = !this.elements.o11ySourcesOptions.classList.contains('hidden');

        if (isOpen) {
            this.closeO11ySourcesDropdown();
        } else {
            this.openO11ySourcesDropdown();
        }
    }

    openO11ySourcesDropdown() {
        this.elements.o11ySourcesOptions.classList.remove('hidden');
        this.elements.o11ySourcesContainer.classList.add('open');
        this.elements.o11ySourcesSearch.focus();

        // Update checkboxes to reflect current selection
        this.updateO11ySourceCheckboxes();
    }

    closeO11ySourcesDropdown() {
        this.elements.o11ySourcesOptions.classList.add('hidden');
        this.elements.o11ySourcesContainer.classList.remove('open');
        this.elements.o11ySourcesSearch.value = '';
        this.filterO11ySources(''); // Show all sources
    }

    toggleO11ySource(source) {
        const index = this.selectedO11ySources.indexOf(source);

        if (index > -1) {
            this.selectedO11ySources.splice(index, 1);
        } else {
            this.selectedO11ySources.push(source);
        }

        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

    updateO11ySourceDisplay() {
        const selectedContainer = this.elements.o11ySourcesSelected;
        const placeholder = this.elements.o11ySourcesPlaceholder;

        // Clear current selection display
        selectedContainer.innerHTML = '';

        if (this.selectedO11ySources.length === 0) {
            placeholder.textContent = 'Select O11y sources...';
            selectedContainer.classList.add('hidden');
        } else {
            placeholder.textContent = `${this.selectedO11ySources.length} source${this.selectedO11ySources.length > 1 ? 's' : ''} selected`;
            selectedContainer.classList.remove('hidden');

            // Add selected items as removable tags
            this.selectedO11ySources.forEach(source => {
                const tag = document.createElement('div');
                tag.className = 'o11y-selected-item';
                tag.innerHTML = `
                    <span>${source}</span>
                    <span class="o11y-selected-item-remove material-symbols-outlined" data-source="${source}">close</span>
                `;

                tag.querySelector('.o11y-selected-item-remove').addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.toggleO11ySource(source);
                });

                selectedContainer.appendChild(tag);
            });
        }
    }

    updateO11ySourceCheckboxes() {
        // Update all checkboxes to reflect current selection
        const checkboxes = this.elements.o11ySourcesList.querySelectorAll('.o11y-source-checkbox');
        checkboxes.forEach(checkbox => {
            const source = checkbox.dataset.source;
            const isSelected = this.selectedO11ySources.includes(source);

            if (isSelected) {
                checkbox.classList.add('checked');
            } else {
                checkbox.classList.remove('checked');
            }
        });
    }

    updateO11ySourceCount() {
        const countElement = this.elements.o11ySourcesCount;
        const totalSources = this.o11ySources.length;
        const selectedCount = this.selectedO11ySources.length;

        countElement.textContent = `${selectedCount}/${totalSources} selected`;
    }

    filterO11ySources(searchTerm) {
        const options = this.elements.o11ySourcesList.querySelectorAll('.o11y-source-option');
        const term = searchTerm.toLowerCase();

        options.forEach(option => {
            const label = option.querySelector('.o11y-source-label');
            const source = label.textContent.toLowerCase();

            if (source.includes(term)) {
                option.style.display = 'flex';
                // Highlight search term
                if (term && source.includes(term)) {
                    const regex = new RegExp(`(${term})`, 'gi');
                    label.innerHTML = label.textContent.replace(regex, '<span class="search-highlight">$1</span>');
                } else {
                    label.innerHTML = label.textContent;
                }
            } else {
                option.style.display = 'none';
            }
        });

        // Show empty state if no results
        const visibleOptions = Array.from(options).filter(opt => opt.style.display !== 'none');
        const emptyState = this.elements.o11ySourcesList.querySelector('.o11y-sources-empty');

        if (visibleOptions.length === 0) {
            if (!emptyState) {
                const empty = document.createElement('div');
                empty.className = 'o11y-sources-empty';
                empty.innerHTML = `
                    <span class="material-symbols-outlined">search_off</span>
                    <p>No sources found matching "${searchTerm}"</p>
                `;
                this.elements.o11ySourcesList.appendChild(empty);
            }
        } else if (emptyState) {
            emptyState.remove();
        }
    }

    selectAllO11ySources() {
        this.selectedO11ySources = [...this.o11ySources];
        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

    clearAllO11ySources() {
        this.selectedO11ySources = [];
        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.vuDataSimManager = new VuDataSimManager();
});

// Test function to manually toggle node management
window.testNodeManagement = function () {
    console.log('Test function called');
    if (window.vuDataSimManager) {
        window.vuDataSimManager.openNodeManagementModal();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Test function to manually load O11y sources
window.testLoadO11ySources = function() {
    console.log('Manual test: Loading O11y sources...');
    if (window.vuDataSimManager) {
        window.vuDataSimManager.loadO11ySources();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Test function to check DOM elements
window.testO11yElements = function() {
    console.log('Testing O11y DOM elements:');
    console.log('- o11y-sources-container:', document.getElementById('o11y-sources-container'));
    console.log('- o11y-sources-dropdown:', document.getElementById('o11y-sources-dropdown'));
    console.log('- o11y-sources-options:', document.getElementById('o11y-sources-options'));
    console.log('- o11y-sources-list:', document.getElementById('o11y-sources-list'));
    console.log('- o11y-sources-search:', document.getElementById('o11y-sources-search'));

    if (window.vuDataSimManager) {
        console.log('vuDataSimManager elements:', window.vuDataSimManager.elements.o11ySourcesList);
        console.log('O11y sources array:', window.vuDataSimManager.o11ySources);
        console.log('Selected sources:', window.vuDataSimManager.selectedO11ySources);
    }
};

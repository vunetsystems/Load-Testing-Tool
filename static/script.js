// vuDataSim Cluster Manager - Frontend JavaScript
class VuDataSimManager {
    constructor() {
        console.log('VuDataSimManager constructor called');
        this.isSimulationRunning = false;
        this.simulationInterval = null;
        this.currentProfile = 'medium';
        this.apiBaseUrl = ''; // Backend API base URL (empty for same origin)
        this.wsConnection = null;
        this.nodes = {}; // Store node data
        
        this.initializeComponents();
        this.bindEvents();
        this.loadNodes(); // Load nodes from API
        this.startRealTimeUpdates();
        console.log('VuDataSimManager initialization complete');
    }

    initializeComponents() {
        console.log('Initializing components...');
        // Cache DOM elements for better performance
        this.elements = {
            profileButton: document.getElementById('profile-button'),
            profileDropdown: document.getElementById('profile-dropdown'),
            selectedProfile: document.getElementById('selected-profile'),
            targetEps: document.getElementById('target-eps'),
            targetKafka: document.getElementById('target-kafka'),
            targetCh: document.getElementById('target-ch'),
            startBtn: document.getElementById('start-btn'),
            stopBtn: document.getElementById('stop-btn'),
            syncBtn: document.getElementById('sync-btn'),
            logNodeFilter: document.getElementById('log-node'),
            logModuleFilter: document.getElementById('log-module'),
            logsContainer: document.getElementById('logs-container'),

            // Node management elements
            nodeManagementBtn: document.getElementById('node-management-btn'),
            nodeManagementSection: document.getElementById('node-management-section'),
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
            
            // Dashboard elements
            nodeStatusIndicators: {
                node1: document.getElementById('node1-status'),
                node2: document.getElementById('node2-status'),
                node3: document.getElementById('node3-status'),
                node4: document.getElementById('node4-status'),
                node5: document.getElementById('node5-status')
            },
            
            // Chart value elements
            epsValue: document.getElementById('eps-value'),
            kafkaValue: document.getElementById('kafka-value'),
            chValue: document.getElementById('ch-value'),
            cpuMemoryValue: document.getElementById('cpu-memory-value'),
            
            // Trend indicators
            epsTrend: document.getElementById('eps-trend'),
            kafkaTrend: document.getElementById('kafka-trend'),
            chTrend: document.getElementById('ch-trend'),
            cpuMemoryTrend: document.getElementById('cpu-memory-trend'),
            
            // Profile summary
            selectedModules: document.getElementById('selected-modules'),
            targetValues: document.getElementById('target-values'),
            projectedEps: document.getElementById('projected-eps'),
            etaRuntime: document.getElementById('eta-runtime')
        };

        // Initialize node data (will be loaded from API)
        this.nodeData = {};

        // Log entries for demonstration
        this.logEntries = [
            { time: '2024-01-20 14:30:00', node: 'Node 1', module: 'Module A', message: 'Starting simulation...', type: 'info' },
            { time: '2024-01-20 14:30:05', node: 'Node 1', module: 'Module A', message: 'Simulation running...', type: 'info' },
            { time: '2024-01-20 14:30:10', node: 'Node 1', module: 'Module A', message: 'EPS: 9800, Kafka: 4900, CH: 1950', type: 'metric' },
            { time: '2024-01-20 14:30:15', node: 'Node 1', module: 'Module A', message: 'EPS: 9900, Kafka: 4950, CH: 1980', type: 'metric' },
            { time: '2024-01-20 14:30:20', node: 'Node 1', module: 'Module A', message: 'EPS: 10100, Kafka: 5050, CH: 2020', type: 'metric' },
            { time: '2024-01-20 14:30:25', node: 'Node 2', module: 'Module B', message: 'Initializing...', type: 'warning' },
            { time: '2024-01-20 14:30:30', node: 'Node 3', module: 'Module A', message: 'Heartbeat OK.', type: 'success' },
            { time: '2024-01-20 14:30:35', node: 'Node 5', module: 'Module A', message: 'Load at 75% capacity.', type: 'info' },
            { time: '2024-01-20 14:30:40', node: 'Node 1', module: 'Module A', message: 'Metric update successful.', type: 'success' },
            { time: '2024-01-20 14:30:45', node: 'Node 4', module: 'System', message: 'Node is Inactive. No logs to display.', type: 'error' }
        ];
    }

    bindEvents() {
        // Profile dropdown functionality
        this.elements.profileButton?.addEventListener('click', (e) => {
            e.stopPropagation();
            this.toggleDropdown();
        });

        // Profile selection
        this.elements.profileDropdown?.querySelectorAll('li').forEach(item => {
            item.addEventListener('click', (e) => {
                const value = e.target.closest('li').dataset.value;
                this.selectProfile(value);
            });
        });

        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('#profile-button') && !e.target.closest('#profile-dropdown')) {
                this.closeDropdown();
            }
        });

        // Button event listeners
        this.elements.startBtn?.addEventListener('click', () => this.startSimulation());
        this.elements.stopBtn?.addEventListener('click', () => this.stopSimulation());
        this.elements.syncBtn?.addEventListener('click', () => this.syncConfiguration());

        // Node management event listeners
        console.log('Node Management Button element:', this.elements.nodeManagementBtn);
        console.log('Node Management Section element:', this.elements.nodeManagementSection);
        
        if (this.elements.nodeManagementBtn) {
            this.elements.nodeManagementBtn.addEventListener('click', () => {
                console.log('Node Management button clicked!');
                this.toggleNodeManagement();
            });
        } else {
            console.error('Node Management button not found!');
        }
        
        this.elements.addNodeBtn?.addEventListener('click', () => this.addNode());

        // Input validation
        [this.elements.targetEps, this.elements.targetKafka, this.elements.targetCh].forEach(input => {
            input?.addEventListener('input', (e) => this.validateInput(e.target));
            input?.addEventListener('blur', (e) => this.formatNumber(e.target));
        });

        // Log filtering
        this.elements.logNodeFilter?.addEventListener('change', () => this.filterLogs());
        this.elements.logModuleFilter?.addEventListener('change', () => this.filterLogs());

        // Real-time updates
        setInterval(() => {
            if (this.isSimulationRunning) {
                this.updateMetrics();
                this.addRandomLog();
            }
        }, 2000);
    }

    async loadNodes() {
        try {
            // Load nodes from the integrated API
            const response = await this.callAPI('/api/nodes');
            if (response.success && response.data) {
                // Convert API data to nodeData format
                this.nodeData = {};
                response.data.forEach(node => {
                    const nodeId = node.name.toLowerCase().replace(/[^a-z0-9]/g, '');
                    this.nodeData[nodeId] = {
                        eps: Math.floor(Math.random() * 10000) + 5000, // Mock metrics for now
                        kafka: Math.floor(Math.random() * 5000) + 2500,
                        ch: Math.floor(Math.random() * 2000) + 1000,
                        cpu: Math.floor(Math.random() * 50) + 30,
                        memory: Math.floor(Math.random() * 50) + 30,
                        status: node.enabled ? 'active' : 'inactive'
                    };
                });

                this.updateDashboardDisplay();
                this.updateNodeStatusIndicators();
            }
        } catch (error) {
            console.error('Error loading nodes:', error);
        }
    }

    toggleDropdown() {
        const dropdown = this.elements.profileDropdown;
        if (dropdown.classList.contains('hidden')) {
            dropdown.classList.remove('hidden');
            dropdown.classList.add('animate-slide-down');
        } else {
            this.closeDropdown();
        }
    }

    closeDropdown() {
        const dropdown = this.elements.profileDropdown;
        dropdown?.classList.add('hidden');
        dropdown?.classList.remove('animate-slide-down');
    }

    selectProfile(profile) {
        this.currentProfile = profile;
        this.elements.selectedProfile.textContent = profile.charAt(0).toUpperCase() + profile.slice(1);
        this.closeDropdown();
        this.updateProfileSummary();
        
        // Show success feedback
        this.showNotification(`Profile changed to ${profile}`, 'success');
    }

    validateInput(input) {
        const value = parseInt(input.value);
        const min = parseInt(input.min);
        const max = parseInt(input.max);

        if (value < min || value > max) {
            input.classList.add('border-danger');
            input.classList.remove('border-primary');
            this.showNotification(`Value must be between ${min} and ${max}`, 'error');
            return false;
        } else {
            input.classList.remove('border-danger');
            input.classList.add('border-primary');
            return true;
        }
    }

    formatNumber(input) {
        const value = parseInt(input.value);
        if (!isNaN(value)) {
            input.value = value.toLocaleString();
        }
    }

    async startSimulation() {
        if (this.isSimulationRunning) {
            this.showNotification('Simulation is already running', 'warning');
            return;
        }

        // Validate inputs before starting
        const inputs = [this.elements.targetEps, this.elements.targetKafka, this.elements.targetCh];
        const allValid = inputs.every(input => this.validateInput(input));

        if (!allValid) {
            this.showNotification('Please fix input validation errors before starting', 'error');
            return;
        }

        // Show loading state
        this.setButtonLoading(this.elements.startBtn, true);

        try {
            // Call backend API to start simulation
            const response = await this.callAPI('/simulation/start', 'POST', {
                profile: this.currentProfile,
                targetEps: parseInt(this.elements.targetEps.value),
                targetKafka: parseInt(this.elements.targetKafka.value),
                targetClickHouse: parseInt(this.elements.targetCh.value)
            });

            if (response.success) {
                this.isSimulationRunning = true;
                this.updateSimulationState();
                this.showNotification('Simulation started successfully', 'success');
                
                // Update projected values
                this.updateProfileSummary();
            } else {
                throw new Error(response.message || 'Failed to start simulation');
            }
        } catch (error) {
            console.error('Error starting simulation:', error);
            this.showNotification('Failed to start simulation: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.elements.startBtn, false);
        }
    }

    async stopSimulation() {
        if (!this.isSimulationRunning) {
            this.showNotification('No simulation is currently running', 'warning');
            return;
        }

        this.setButtonLoading(this.elements.stopBtn, true);

        try {
            const response = await this.callAPI('/simulation/stop', 'POST');

            if (response.success) {
                this.isSimulationRunning = false;
                this.updateSimulationState();
                this.showNotification('Simulation stopped successfully', 'success');
            } else {
                throw new Error(response.message || 'Failed to stop simulation');
            }
        } catch (error) {
            console.error('Error stopping simulation:', error);
            this.showNotification('Failed to stop simulation: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.elements.stopBtn, false);
        }
    }

    async syncConfiguration() {
        this.setButtonLoading(this.elements.syncBtn, true);

        try {
            const response = await this.callAPI('/config/sync', 'POST');

            if (response.success) {
                this.showNotification('Configuration synced successfully', 'success');
                // Update dashboard with fresh data
                this.refreshDashboard();
            } else {
                throw new Error(response.message || 'Failed to sync configuration');
            }
        } catch (error) {
            console.error('Error syncing configuration:', error);
            this.showNotification('Failed to sync configuration: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.elements.syncBtn, false);
        }
    }

    setButtonLoading(button, isLoading) {
        if (isLoading) {
            button.classList.add('loading');
            button.disabled = true;
            const originalText = button.textContent;
            button.dataset.originalText = originalText;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Processing...</span>';
        } else {
            button.classList.remove('loading');
            button.disabled = false;
            if (button.dataset.originalText) {
                button.innerHTML = button.dataset.originalText;
            }
        }
    }

    updateSimulationState() {
        if (this.isSimulationRunning) {
            this.elements.startBtn.disabled = true;
            this.elements.startBtn.classList.add('opacity-50');
            this.elements.stopBtn.disabled = false;
            this.elements.stopBtn.classList.remove('opacity-50');
        } else {
            this.elements.startBtn.disabled = false;
            this.elements.startBtn.classList.remove('opacity-50');
            this.elements.stopBtn.disabled = true;
            this.elements.stopBtn.classList.add('opacity-50');
        }
    }

    updateMetrics() {
        // Simulate real-time metric updates
        Object.keys(this.nodeData).forEach(nodeId => {
            const node = this.nodeData[nodeId];
            if (node.status === 'active') {
                // Add small random variations
                const variation = (Math.random() - 0.5) * 200;
                node.eps = Math.max(0, node.eps + variation);
                node.kafka = Math.max(0, node.kafka + variation / 2);
                node.ch = Math.max(0, node.ch + variation / 5);
                
                // Update CPU and memory with smaller variations
                node.cpu = Math.max(0, Math.min(100, node.cpu + (Math.random() - 0.5) * 5));
                node.memory = Math.max(0, Math.min(100, node.memory + (Math.random() - 0.5) * 5));
            }
        });

        this.updateDashboardDisplay();
    }

    updateDashboardDisplay() {
        // Update cluster table
        Object.keys(this.nodeData).forEach(nodeId => {
            const node = this.nodeData[nodeId];
            const row = document.querySelector(`tr:nth-child(${parseInt(nodeId.slice(-1)) + (nodeId === 'node4' ? 0 : 1)})`);
            
            if (row) {
                const cells = row.querySelectorAll('[data-field]');
                cells.forEach(cell => {
                    const field = cell.dataset.field;
                    const value = node[field];
                    const formattedValue = field === 'cpu' || field === 'memory' ? 
                        `${Math.round(value)}%` : 
                        Math.round(value).toLocaleString();
                    cell.textContent = formattedValue;
                });
            }
        });

        // Update chart values
        const totalEps = Object.values(this.nodeData).reduce((sum, node) => sum + node.eps, 0);
        const totalKafka = Object.values(this.nodeData).reduce((sum, node) => sum + node.kafka, 0);
        const totalCh = Object.values(this.nodeData).reduce((sum, node) => sum + node.ch, 0);
        
        this.animateNumber(this.elements.epsValue, totalEps);
        this.animateNumber(this.elements.kafkaValue, totalKafka);
        this.animateNumber(this.elements.chValue, totalCh);
        
        // Calculate averages for CPU/Memory
        const activeNodes = Object.values(this.nodeData).filter(node => node.status === 'active');
        const avgCpu = activeNodes.reduce((sum, node) => sum + node.cpu, 0) / activeNodes.length;
        const avgMemory = activeNodes.reduce((sum, node) => sum + node.memory, 0) / activeNodes.length;
        
        this.elements.cpuMemoryValue.textContent = `${Math.round(avgCpu)}% / ${Math.round(avgMemory)}%`;
        
        // Update trends (simulated)
        this.updateTrend(this.elements.epsTrend, Math.random() > 0.5 ? 'up' : 'down');
        this.updateTrend(this.elements.kafkaTrend, Math.random() > 0.6 ? 'up' : 'down');
        this.updateTrend(this.elements.chTrend, Math.random() > 0.5 ? 'up' : 'down');
        this.updateTrend(this.elements.cpuMemoryTrend, Math.random() > 0.5 ? 'up' : 'down');
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

    updateProfileSummary() {
        const targetEps = parseInt(this.elements.targetEps.value) || 10000;
        const targetKafka = parseInt(this.elements.targetKafka.value) || 5000;
        const targetCh = parseInt(this.elements.targetCh.value) || 2000;
        
        this.elements.targetValues.textContent = `EPS: ${targetEps.toLocaleString()}, Kafka: ${targetKafka.toLocaleString()}, CH: ${targetCh.toLocaleString()}`;
        
        // Calculate projected EPS based on current performance
        const currentTotalEps = Object.values(this.nodeData).reduce((sum, node) => sum + node.eps, 0);
        const projectedEps = Math.round(currentTotalEps * 0.98); // 98% efficiency projection
        this.animateNumber(this.elements.projectedEps, projectedEps);
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
        const activeNodes = Object.keys(this.nodeData).filter(id => this.nodeData[id].status === 'active');
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
        // Initial display
        this.updateDashboardDisplay();
        this.updateNodeStatusIndicators();
        this.refreshNodesTable();
        this.updateProfileSummary();
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
            // Also refresh the nodes table if it's visible
            if (!this.elements.nodeManagementSection.classList.contains('hidden')) {
                this.refreshNodesTable();
            }
        } catch (error) {
            console.error('Error refreshing dashboard:', error);
        }
    }

    updateNodeStatusIndicators() {
        // Update node status indicators based on real node data
        Object.keys(this.nodeData).forEach((nodeId, index) => {
            const node = this.nodeData[nodeId];
            const statusElement = this.elements.nodeStatusIndicators[`node${index + 1}`];

            if (statusElement) {
                if (node.status === 'active') {
                    statusElement.className = 'h-4 w-4 rounded-full bg-success animate-node-pulse';
                    statusElement.title = `${nodeId.charAt(0).toUpperCase() + nodeId.slice(1)}: Active`;
                } else {
                    statusElement.className = 'h-4 w-4 rounded-full bg-danger';
                    statusElement.title = `${nodeId.charAt(0).toUpperCase() + nodeId.slice(1)}: Inactive`;
                }
            }
        });
    }

    toggleNodeManagement() {
        console.log('toggleNodeManagement called');
        const section = this.elements.nodeManagementSection;
        console.log('Node management section:', section);
        
        if (!section) {
            console.error('Node management section not found!');
            return;
        }
        
        const isHidden = section.classList.contains('hidden');
        console.log('Section is hidden:', isHidden);

        if (isHidden) {
            console.log('Showing node management section');
            section.classList.remove('hidden');
            this.refreshNodesTable();
        } else {
            console.log('Hiding node management section');
            section.classList.add('hidden');
        }
    }

    async addNode() {
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

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 animate-slide-down ${
            type === 'success' ? 'bg-success text-white' :
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
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.vuDataSimManager = new VuDataSimManager();
});

// Test function to manually toggle node management
window.testNodeManagement = function() {
    console.log('Test function called');
    if (window.vuDataSimManager) {
        window.vuDataSimManager.toggleNodeManagement();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Export for potential module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = VuDataSimManager;
}
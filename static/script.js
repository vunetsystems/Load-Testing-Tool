// vuDataSim Cluster Manager - Frontend JavaScript
class VuDataSimManager {
    constructor() {
        this.isSimulationRunning = false;
        this.simulationInterval = null;
        this.currentProfile = 'medium';
        this.apiBaseUrl = '/api'; // Backend API base URL
        this.wsConnection = null;
        
        this.initializeComponents();
        this.bindEvents();
        this.startRealTimeUpdates();
    }

    initializeComponents() {
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

        // Initialize node data
        this.nodeData = {
            node1: { eps: 9800, kafka: 4900, ch: 1950, cpu: 65, memory: 70, status: 'active' },
            node2: { eps: 9900, kafka: 4950, ch: 1980, cpu: 70, memory: 75, status: 'active' },
            node3: { eps: 10100, kafka: 5050, ch: 2020, cpu: 60, memory: 65, status: 'active' },
            node4: { eps: 0, kafka: 0, ch: 0, cpu: 0, memory: 0, status: 'inactive' },
            node5: { eps: 10000, kafka: 5000, ch: 2000, cpu: 75, memory: 80, status: 'active' }
        };

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
            const response = await this.callAPI('/dashboard/refresh');
            if (response.success && response.data) {
                // Update with fresh data from backend
                Object.assign(this.nodeData, response.data.nodes || {});
                this.updateDashboardDisplay();
            }
        } catch (error) {
            console.error('Error refreshing dashboard:', error);
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

// Export for potential module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = VuDataSimManager;
}
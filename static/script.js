// vuDataSim Cluster Manager - Frontend JavaScript
class VuDataSimManager {
    constructor() {
        console.log('VuDataSimManager constructor called');
        this.isSimulationRunning = false;
        this.apiBaseUrl = ''; // Backend API base URL (empty for same origin)
        this.wsConnection = null;
        this.nodes = {}; // Store node data
        
        this.initializeComponents();
        this.bindEvents();
        this.loadNodes(); // Load nodes from API
        this.startRealTimeUpdates();
        this.populateNodeFilters(); // Populate filter dropdowns with real node names
        console.log('VuDataSimManager initialization complete');
    }

    initializeComponents() {
        console.log('Initializing components...');
        // Cache DOM elements for better performance
        this.elements = {
            syncBtn: document.getElementById('sync-btn'),
            logNodeFilter: document.getElementById('log-node'),
            logModuleFilter: document.getElementById('log-module'),
            logsContainer: document.getElementById('logs-container'),

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

            // Dashboard elements
            nodeStatusIndicatorsContainer: document.getElementById('node-status-indicators'),

            // Chart value elements
            cpuMemoryValue: document.getElementById('cpu-memory-value'),

            // Real-time status
        };

        // Initialize node data (will be loaded from API with real data only)
        this.nodeData = {};

        // Legacy property for backward compatibility (browser cache)
        this.nodeManagementSection = null;

        // Real-time log entries (will be loaded from API)
        this.logEntries = [];
        this.lastLogUpdate = 0;
    }

    bindEvents() {
        // Button event listeners
        this.elements.syncBtn?.addEventListener('click', () => this.refreshRealData());

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

        // Modal event listeners
        this.elements.closeNodeModal?.addEventListener('click', () => this.closeNodeManagementModal());
        this.elements.modalBackdrop?.addEventListener('click', () => this.closeNodeManagementModal());

        // Close modal on Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && !this.elements.nodeManagementModal.classList.contains('hidden')) {
                this.closeNodeManagementModal();
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
        }, 3000);

        // Load logs initially and set up polling for live logs
        this.loadLogs();
        setInterval(() => {
            this.loadLogs();
        }, 5000); // Update logs every 5 seconds
    }

    async loadNodes() {
        try {
            // Load nodes from the integrated API
            const response = await this.callAPI('/api/dashboard');
            if (response.success && response.data && response.data.nodeData) {
                this.nodeData = response.data.nodeData;
                this.updateDashboardDisplay();
                this.updateNodeStatusIndicators();
            } else {
                // Fallback: load from nodes API if dashboard doesn't have nodeData
                const nodesResponse = await this.callAPI('/api/nodes');
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
                            status: node.enabled ? 'active' : 'inactive'
                        };
                    });

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

        // Add actual node names
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
        this.updateDashboardDisplay();
    }

    updateDashboardDisplay() {
        // Update cluster table dynamically
        const tbody = document.getElementById('cluster-table-body');
        tbody.innerHTML = ''; // Clear existing rows

        Object.keys(this.nodeData).forEach(nodeId => {
            const node = this.nodeData[nodeId];
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            if (node.status === 'inactive') {
                row.classList.add('opacity-60');
            }

            // Calculate memory usage in GB
            const usedMemory = node.totalMemory * (node.memory / 100);

            row.innerHTML = `
                <td class="p-4 font-medium">${nodeId}</td>
                <td class="p-4">
                    <div class="inline-flex items-center gap-2 rounded-full ${node.status === 'active' ? 'bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark' : 'bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark'}">
                        <span class="h-2 w-2 rounded-full ${node.status === 'active' ? 'bg-success' : 'bg-danger'}"></span>${node.status === 'active' ? 'Active' : 'Inactive'}
                    </div>
                </td>
                <td class="p-4 text-right number-animate" data-field="cpu">${(node.totalCpu - (node.totalCpu * node.cpu / 100)).toFixed(1)} / ${node.totalCpu} cores</td>
                <td class="p-4 text-right number-animate" data-field="memory">${usedMemory.toFixed(1)} / ${node.totalMemory} GB</td>
            `;

            tbody.appendChild(row);
        });

        // Calculate real-time CPU/Memory data only
        const activeNodes = Object.values(this.nodeData).filter(node => node.status === 'active');
        if (activeNodes.length > 0) {
            const totalAvgCpu = activeNodes.reduce((sum, node) => sum + node.totalCpu, 0) / activeNodes.length;
            const totalAvgMemory = activeNodes.reduce((sum, node) => sum + node.totalMemory, 0) / activeNodes.length;

            const avgCpuUsage = activeNodes.reduce((sum, node) => sum + node.cpu, 0) / activeNodes.length;
            const avgMemoryUsage = activeNodes.reduce((sum, node) => sum + node.memory, 0) / activeNodes.length;

            const availableCpu = totalAvgCpu - (totalAvgCpu * avgCpuUsage / 100);
            const usedMemory = totalAvgMemory * (avgMemoryUsage / 100);

            this.elements.cpuMemoryValue.textContent = `${availableCpu.toFixed(1)}/${totalAvgCpu.toFixed(1)} cores / ${usedMemory.toFixed(1)}/${totalAvgMemory.toFixed(1)} GB`;
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
        window.vuDataSimManager.openNodeManagementModal();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Export for potential module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = VuDataSimManager;
}
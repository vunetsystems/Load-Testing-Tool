// Cluster Metrics JavaScript Module

class ClusterMetricsManager {
    constructor() {
        this.clusterData = null;
        this.lastUpdate = null;
        this.updateInterval = 30000; // 30 seconds
        this.isUpdating = false;
    }

    // Initialize cluster metrics
    async initialize() {
        console.log('Initializing cluster metrics...');
        await this.fetchClusterMetrics();
        this.startAutoUpdate();
    }

    // Fetch cluster metrics from API
    async fetchClusterMetrics() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            const response = await fetch('/api/clickhouse/cluster-metrics');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.clusterData = result.data;
                this.lastUpdate = new Date();
                console.log('Cluster metrics updated:', this.clusterData);
                this.updateClusterDisplay();
            } else {
                console.error('Failed to fetch cluster metrics:', result.message);
            }
        } catch (error) {
            console.error('Error fetching cluster metrics:', error);
        } finally {
            this.isUpdating = false;
        }
    }

    // Update cluster metrics display
    updateClusterDisplay() {
        if (!this.clusterData) return;

        // Update node status table
        this.updateNodeStatusTable();

        // Update quick stats
        this.updateQuickStats();

        // Update cluster tab if visible
        this.updateClusterTab();

        // Update charts if they exist
        this.updateCharts();
    }

    // Update node status table
    updateNodeStatusTable() {
        const tbody = document.getElementById('node-status-body');
        if (!tbody || !this.clusterData) return;

        // Clear existing rows
        tbody.innerHTML = '';

        // Add rows for each node
        Object.entries(this.clusterData).forEach(([nodeName, metrics]) => {
            const row = document.createElement('tr');

            // Determine status based on CPU and memory usage
            const cpuUsage = ((metrics.CPUCores * 100) / 8); // Assuming 8 cores per node
            const memoryUsage = (metrics.UsedMemoryGB / metrics.TotalMemoryGB) * 100;
            let status = 'Online';
            let statusClass = 'success';

            if (cpuUsage > 90 || memoryUsage > 90) {
                status = 'Warning';
                statusClass = 'warning';
            } else if (cpuUsage > 75 || memoryUsage > 75) {
                status = 'Warning';
                statusClass = 'warning';
            }

            row.innerHTML = `
                <td class="p-4">${nodeName}</td>
                <td class="p-4">
                    <div class="flex items-center gap-2">
                        <span class="h-2 w-2 rounded-full bg-${statusClass}"></span>
                        <span class="text-${statusClass} font-medium">${status}</span>
                    </div>
                </td>
                <td class="p-4">${cpuUsage.toFixed(1)}%</td>
                <td class="p-4">${metrics.UsedMemoryGB.toFixed(1)} GB / ${metrics.TotalMemoryGB.toFixed(1)} GB</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">${this.lastUpdate ? this.lastUpdate.toLocaleTimeString() : 'Just now'}</td>
            `;

            tbody.appendChild(row);
        });
    }

    // Update quick stats
    updateQuickStats() {
        if (!this.clusterData) return;

        const nodeCount = Object.keys(this.clusterData).length;
        const totalCores = Object.values(this.clusterData).reduce((sum, node) => sum + node.CPUCores, 0);
        const totalMemory = Object.values(this.clusterData).reduce((sum, node) => sum + node.TotalMemoryGB, 0);
        const usedMemory = Object.values(this.clusterData).reduce((sum, node) => sum + node.UsedMemoryGB, 0);
        const avgCpuUsage = totalCores > 0 ? (Object.values(this.clusterData).reduce((sum, node) => sum + node.CPUCores, 0) / nodeCount) : 0;

        // Update active nodes
        const activeNodesElement = document.querySelector('[data-stat="active-nodes"]') ||
                                  document.querySelector('.quick-stats .grid > div:nth-child(2) p:nth-child(2)');
        if (activeNodesElement) {
            activeNodesElement.textContent = `${nodeCount}/${nodeCount}`;
        }

        // Update avg CPU
        const avgCpuElement = document.querySelector('[data-stat="avg-cpu"]') ||
                             document.querySelector('.quick-stats .grid > div:nth-child(3) p:nth-child(2)');
        if (avgCpuElement) {
            avgCpuElement.textContent = `${avgCpuUsage.toFixed(1)}%`;
        }

        // Update memory usage
        const memoryElement = document.querySelector('[data-stat="memory-usage"]') ||
                             document.querySelector('.quick-stats .grid > div:nth-child(4) p:nth-child(2)');
        if (memoryElement) {
            memoryElement.textContent = `${usedMemory.toFixed(1)}GB`;
        }
    }

    // Update cluster tab display
    updateClusterTab() {
        if (!this.clusterData) return;

        // Update cluster summary cards
        this.updateClusterSummaryCards();

        // Update cluster node status table in performance tab
        this.updateClusterNodeStatusTable();
    }

    // Update cluster summary cards
    updateClusterSummaryCards() {
        const nodeCount = Object.keys(this.clusterData).length;
        const totalCores = Object.values(this.clusterData).reduce((sum, node) => sum + node.CPUCores, 0);
        const totalMemory = Object.values(this.clusterData).reduce((sum, node) => sum + node.TotalMemoryGB, 0);
        const usedMemory = Object.values(this.clusterData).reduce((sum, node) => sum + node.UsedMemoryGB, 0);
        const avgCpuUsage = totalCores > 0 ? (Object.values(this.clusterData).reduce((sum, node) => sum + node.CPUCores, 0) / nodeCount) : 0;

        // Update total nodes
        const totalNodesElement = document.getElementById('cluster-total-nodes');
        if (totalNodesElement) {
            totalNodesElement.textContent = nodeCount;
        }

        // Update avg CPU
        const avgCpuElement = document.getElementById('cluster-avg-cpu');
        if (avgCpuElement) {
            avgCpuElement.textContent = `${avgCpuUsage.toFixed(1)}%`;
        }

        // Update total memory
        const totalMemoryElement = document.getElementById('cluster-total-memory');
        if (totalMemoryElement) {
            totalMemoryElement.textContent = `${totalMemory.toFixed(1)} GB`;
        }

        // Update used memory
        const usedMemoryElement = document.getElementById('cluster-used-memory');
        if (usedMemoryElement) {
            usedMemoryElement.textContent = `${usedMemory.toFixed(1)} GB`;
        }
    }

    // Update cluster node status table in performance tab
    updateClusterNodeStatusTable() {
        const tbody = document.getElementById('cluster-node-status-body');
        if (!tbody || !this.clusterData) return;

        // Clear existing rows
        tbody.innerHTML = '';

        // Add rows for each node
        Object.entries(this.clusterData).forEach(([nodeName, metrics]) => {
            const row = document.createElement('tr');

            // Determine status based on CPU and memory usage
            const cpuUsage = ((metrics.CPUCores * 100) / 8); // Assuming 8 cores per node
            const memoryUsage = (metrics.UsedMemoryGB / metrics.TotalMemoryGB) * 100;
            let status = 'Online';
            let statusClass = 'success';

            if (cpuUsage > 90 || memoryUsage > 90) {
                status = 'Warning';
                statusClass = 'warning';
            } else if (cpuUsage > 75 || memoryUsage > 75) {
                status = 'Warning';
                statusClass = 'warning';
            }

            row.innerHTML = `
                <td class="p-4">${nodeName}</td>
                <td class="p-4">
                    <div class="flex items-center gap-2">
                        <span class="h-2 w-2 rounded-full bg-${statusClass}"></span>
                        <span class="text-${statusClass} font-medium">${status}</span>
                    </div>
                </td>
                <td class="p-4">${cpuUsage.toFixed(1)}%</td>
                <td class="p-4">${metrics.UsedMemoryGB.toFixed(1)} GB / ${metrics.TotalMemoryGB.toFixed(1)} GB</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">${this.lastUpdate ? this.lastUpdate.toLocaleTimeString() : 'Just now'}</td>
            `;

            tbody.appendChild(row);
        });
    }

    // Update charts (placeholder for future chart implementation)
    updateCharts() {
        // This would update Chart.js or other charting libraries
        console.log('Charts updated with cluster data');
    }

    // Start automatic updates
    startAutoUpdate() {
        setInterval(() => {
            this.fetchClusterMetrics();
        }, this.updateInterval);
    }

    // Manual refresh
    async refresh() {
        await this.fetchClusterMetrics();
    }

    // Get current metrics data
    getMetrics() {
        return {
            data: this.clusterData,
            lastUpdate: this.lastUpdate,
            nodeCount: this.clusterData ? Object.keys(this.clusterData).length : 0
        };
    }
}

// Export for use in other modules
window.ClusterMetricsManager = ClusterMetricsManager;
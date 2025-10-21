// System Health JavaScript Module

class SystemHealthManager {
    constructor() {
        this.healthData = null;
        this.podMetrics = null;
        this.lastUpdate = null;
        this.updateInterval = 30000; // 30 seconds
        this.isUpdating = false;
    }

    // Initialize system health monitoring
    async initialize() {
        console.log('Initializing system health monitoring...');
        await this.fetchSystemHealth();
        await this.fetchPodMetrics();
        this.startAutoUpdate();
    }

    // Fetch system health from API
    async fetchSystemHealth() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            const response = await fetch('/api/clickhouse/health');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.healthData = result.data;
                console.log('System health updated:', this.healthData);
                this.updateSystemHealthDisplay();
            } else {
                console.error('Failed to fetch system health:', result.message);
            }
        } catch (error) {
            console.error('Error fetching system health:', error);
        } finally {
            this.isUpdating = false;
        }
    }

    // Fetch pod metrics from API
    async fetchPodMetrics() {
        try {
            const response = await fetch('/api/clickhouse/pod-metrics');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.podMetrics = result.data;
                console.log('Pod metrics updated:', this.podMetrics);
                this.updatePodMetricsDisplay();
            } else {
                console.error('Failed to fetch pod metrics:', result.message);
            }
        } catch (error) {
            console.error('Error fetching pod metrics:', error);
        }
    }

    // Update system health display
    updateSystemHealthDisplay() {
        if (!this.healthData) return;

        // Update system status indicator
        this.updateSystemStatus();

        // Update connection details
        this.updateConnectionDetails();

        // Update health cards
        this.updateHealthCards();

        // Update system health tab if visible
        this.updateSystemHealthTab();
    }

    // Update system status indicator
    updateSystemStatus() {
        const status = this.healthData.status;
        const statusElement = document.querySelector('.quick-stats .grid > div:nth-child(1) p:nth-child(2)');

        if (statusElement) {
            let statusText = 'Unknown';
            let statusClass = 'text-text-secondary-light dark:text-text-secondary-dark';

            switch (status) {
                case 'connected':
                    statusText = 'Healthy';
                    statusClass = 'text-success dark:text-success-dark';
                    break;
                case 'disconnected':
                    statusText = 'Disconnected';
                    statusClass = 'text-danger dark:text-danger-dark';
                    break;
                case 'error':
                    statusText = 'Error';
                    statusClass = 'text-warning dark:text-warning-dark';
                    break;
            }

            statusElement.textContent = statusText;
            statusElement.className = statusElement.className.replace(/text-\w+/g, '') + ' ' + statusClass;
        }
    }

    // Update connection details
    updateConnectionDetails() {
        const systemSection = document.getElementById('system-section');
        if (!systemSection || !this.healthData) return;

        // Create or update connection details section
        let detailsDiv = systemSection.querySelector('.connection-details');
        if (!detailsDiv) {
            detailsDiv = document.createElement('div');
            detailsDiv.className = 'connection-details grid grid-cols-1 md:grid-cols-2 gap-6 mb-8';
            systemSection.appendChild(detailsDiv);
        }

        detailsDiv.innerHTML = `
            <div class="rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark p-6 shadow-md">
                <h3 class="text-lg font-semibold mb-4">ClickHouse Connection</h3>
                <div class="space-y-3">
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Status</span>
                        <span class="font-medium ${this.healthData.status === 'connected' ? 'text-success dark:text-success-dark' : 'text-danger dark:text-danger-dark'}">
                            ${this.healthData.status === 'connected' ? 'Connected' : 'Disconnected'}
                        </span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Host</span>
                        <span class="font-medium">${this.healthData.host || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Port</span>
                        <span class="font-medium">${this.healthData.port || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Database</span>
                        <span class="font-medium">${this.healthData.database || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Last Checked</span>
                        <span class="font-medium">${this.healthData.last_checked ? new Date(this.healthData.last_checked).toLocaleString() : 'Never'}</span>
                    </div>
                </div>
            </div>
        `;

        // Add error details if present
        if (this.healthData.error) {
            const errorCard = document.createElement('div');
            errorCard.className = 'rounded-xl border border-danger/30 dark:border-danger-dark/30 bg-danger/10 dark:bg-danger-dark/10 p-6 shadow-md';
            errorCard.innerHTML = `
                <h3 class="text-lg font-semibold mb-4 text-danger dark:text-danger-dark">Connection Error</h3>
                <p class="text-sm">${this.healthData.error}</p>
            `;
            detailsDiv.appendChild(errorCard);
        }
    }

    // Update health cards
    updateHealthCards() {
        // Update any health-related cards in the overview
        const systemLoadElement = document.querySelector('[data-stat="system-load"]') ||
                                 document.querySelector('.grid > div:nth-child(3) p:nth-child(2)');

        if (systemLoadElement && this.healthData.status === 'connected') {
            // Simulate system load (in a real app, this would come from actual metrics)
            const loadValue = (Math.random() * 2 + 1).toFixed(1);
            systemLoadElement.textContent = loadValue;
        }
    }

    // Update pod metrics display
    updatePodMetricsDisplay() {
        if (!this.podMetrics) return;

        const systemSection = document.getElementById('system-section');
        if (!systemSection) return;

        // Create pod metrics section
        this.createPodResourceMetrics();
        this.createPodStatusMetrics();
        this.createTopMemoryPods();
    }

    // Create pod resource metrics section
    createPodResourceMetrics() {
        if (!this.podMetrics.podResourceMetrics || this.podMetrics.podResourceMetrics.length === 0) return;

        const systemSection = document.getElementById('system-section');
        let resourceSection = systemSection.querySelector('.pod-resource-metrics');

        if (!resourceSection) {
            resourceSection = document.createElement('div');
            resourceSection.className = 'pod-resource-metrics';
            systemSection.appendChild(resourceSection);
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Pod Resource Utilization';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">CPU Usage</th>
                    <th class="p-4 font-semibold">Memory Usage</th>
                    <th class="p-4 font-semibold">Last Update</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.podResourceMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.cpuPercentage.toFixed(1)}%</td>
                <td class="p-4">${metric.memoryPercentage.toFixed(1)}%</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(metric.lastTimestamp).toLocaleTimeString()}
                </td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        resourceSection.innerHTML = '';
        resourceSection.appendChild(title);
        resourceSection.appendChild(table);
    }

    // Create pod status metrics section
    createPodStatusMetrics() {
        if (!this.podMetrics.podStatusMetrics || this.podMetrics.podStatusMetrics.length === 0) return;

        const systemSection = document.getElementById('system-section');
        let statusSection = systemSection.querySelector('.pod-status-metrics');

        if (!statusSection) {
            statusSection = document.createElement('div');
            statusSection.className = 'pod-status-metrics mt-8';
            systemSection.appendChild(statusSection);
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Pod Status Overview';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">Node</th>
                    <th class="p-4 font-semibold">Phase</th>
                    <th class="p-4 font-semibold">Status</th>
                    <th class="p-4 font-semibold">Running Containers</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.podStatusMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.nodeName}</td>
                <td class="p-4">${metric.podPhase}</td>
                <td class="p-4">
                    <span class="px-2 py-1 rounded-full text-xs font-medium
                        ${metric.derivedStatus === 'Running' ? 'bg-success/20 text-success dark:bg-success-dark/20 dark:text-success-dark' :
                          metric.derivedStatus === 'Pending' ? 'bg-warning/20 text-warning dark:bg-warning-dark/20 dark:text-warning-dark' :
                          'bg-danger/20 text-danger dark:bg-danger-dark/20 dark:text-danger-dark'}">
                        ${metric.derivedStatus}
                    </span>
                </td>
                <td class="p-4">${metric.runningContainers}/${metric.runningContainers + metric.nonRunningContainers}</td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        statusSection.innerHTML = '';
        statusSection.appendChild(title);
        statusSection.appendChild(table);
    }

    // Create top memory pods section
    createTopMemoryPods() {
        if (!this.podMetrics.topPodMemoryMetrics || this.podMetrics.topPodMemoryMetrics.length === 0) return;

        const systemSection = document.getElementById('system-section');
        let memorySection = systemSection.querySelector('.top-memory-pods');

        if (!memorySection) {
            memorySection = document.createElement('div');
            memorySection.className = 'top-memory-pods mt-8';
            systemSection.appendChild(memorySection);
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Top Memory Usage by Pod';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Node</th>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">Memory Usage</th>
                    <th class="p-4 font-semibold">Last Update</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.topPodMemoryMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.nodeIp}</td>
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.memoryPct.toFixed(1)}%</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(metric.timestamp).toLocaleTimeString()}
                </td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        memorySection.innerHTML = '';
        memorySection.appendChild(title);
        memorySection.appendChild(table);
    }

    // Update system health tab display
    updateSystemHealthTab() {
        if (!this.healthData && !this.podMetrics) return;

        // Update system health summary cards
        this.updateSystemHealthSummaryCards();

        // Update pod metrics in performance tab
        this.updatePodMetricsInTab();
    }

    // Update system health summary cards
    updateSystemHealthSummaryCards() {
        // Update ClickHouse status
        const statusElement = document.getElementById('system-clickhouse-status');
        if (statusElement && this.healthData) {
            const status = this.healthData.status === 'connected' ? 'Connected' : 'Disconnected';
            const statusClass = this.healthData.status === 'connected' ? 'text-success dark:text-success-dark' : 'text-danger dark:text-danger-dark';
            statusElement.textContent = status;
            statusElement.className = statusElement.className.replace(/text-\w+/g, '') + ' ' + statusClass;
        }

        // Update pod counts if available
        if (this.podMetrics && this.podMetrics.podStatusMetrics) {
            const totalPods = this.podMetrics.podStatusMetrics.length;
            const runningPods = this.podMetrics.podStatusMetrics.filter(pod => pod.derivedStatus === 'Running').length;
            const failedPods = this.podMetrics.podStatusMetrics.filter(pod => pod.derivedStatus !== 'Running').length;

            // Update total pods
            const totalPodsElement = document.getElementById('system-total-pods');
            if (totalPodsElement) {
                totalPodsElement.textContent = totalPods;
            }

            // Update running pods
            const runningPodsElement = document.getElementById('system-running-pods');
            if (runningPodsElement) {
                runningPodsElement.textContent = runningPods;
            }

            // Update failed pods
            const failedPodsElement = document.getElementById('system-failed-pods');
            if (failedPodsElement) {
                failedPodsElement.textContent = failedPods;
            }
        }
    }

    // Update pod metrics in performance tab
    updatePodMetricsInTab() {
        if (!this.podMetrics) return;

        // Update connection details in performance tab
        this.updateConnectionDetailsInTab();

        // Update pod resource metrics in performance tab
        this.updatePodResourceMetricsInTab();

        // Update pod status metrics in performance tab
        this.updatePodStatusMetricsInTab();

        // Update top memory pods in performance tab
        this.updateTopMemoryPodsInTab();
    }

    // Update connection details in performance tab
    updateConnectionDetailsInTab() {
        const tabContainer = document.getElementById('system-tab');
        if (!tabContainer || !this.healthData) return;

        const detailsContainer = tabContainer.querySelector('.connection-details');
        if (!detailsContainer) return;

        detailsContainer.innerHTML = `
            <div class="rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark p-6 shadow-md">
                <h3 class="text-lg font-semibold mb-4">ClickHouse Connection</h3>
                <div class="space-y-3">
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Status</span>
                        <span class="font-medium ${this.healthData.status === 'connected' ? 'text-success dark:text-success-dark' : 'text-danger dark:text-danger-dark'}">
                            ${this.healthData.status === 'connected' ? 'Connected' : 'Disconnected'}
                        </span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Host</span>
                        <span class="font-medium">${this.healthData.host || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Port</span>
                        <span class="font-medium">${this.healthData.port || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Database</span>
                        <span class="font-medium">${this.healthData.database || 'N/A'}</span>
                    </div>
                    <div class="flex justify-between">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Last Checked</span>
                        <span class="font-medium">${this.healthData.last_checked ? new Date(this.healthData.last_checked).toLocaleString() : 'Never'}</span>
                    </div>
                </div>
            </div>
        `;

        // Add error details if present
        if (this.healthData.error) {
            const errorCard = document.createElement('div');
            errorCard.className = 'rounded-xl border border-danger/30 dark:border-danger-dark/30 bg-danger/10 dark:bg-danger-dark/10 p-6 shadow-md';
            errorCard.innerHTML = `
                <h3 class="text-lg font-semibold mb-4 text-danger dark:text-danger-dark">Connection Error</h3>
                <p class="text-sm">${this.healthData.error}</p>
            `;
            detailsContainer.appendChild(errorCard);
        }
    }

    // Update pod resource metrics in performance tab
    updatePodResourceMetricsInTab() {
        const tabContainer = document.getElementById('system-tab');
        if (!tabContainer || !this.podMetrics || !this.podMetrics.podResourceMetrics) return;

        const resourceContainer = tabContainer.querySelector('.pod-resource-metrics');
        if (!resourceContainer) return;

        if (this.podMetrics.podResourceMetrics.length === 0) {
            resourceContainer.innerHTML = `
                <h3 class="text-xl font-semibold mb-4">Pod Resource Utilization</h3>
                <p class="text-text-secondary-light dark:text-text-secondary-dark">No pod resource metrics available</p>
            `;
            return;
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Pod Resource Utilization';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">CPU Usage</th>
                    <th class="p-4 font-semibold">Memory Usage</th>
                    <th class="p-4 font-semibold">Last Update</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.podResourceMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.cpuPercentage.toFixed(1)}%</td>
                <td class="p-4">${metric.memoryPercentage.toFixed(1)}%</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(metric.lastTimestamp).toLocaleTimeString()}
                </td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        resourceContainer.innerHTML = '';
        resourceContainer.appendChild(title);
        resourceContainer.appendChild(table);
    }

    // Update pod status metrics in performance tab
    updatePodStatusMetricsInTab() {
        const tabContainer = document.getElementById('system-tab');
        if (!tabContainer || !this.podMetrics || !this.podMetrics.podStatusMetrics) return;

        const statusContainer = tabContainer.querySelector('.pod-status-metrics');
        if (!statusContainer) return;

        if (this.podMetrics.podStatusMetrics.length === 0) {
            statusContainer.innerHTML = `
                <h3 class="text-xl font-semibold mb-4">Pod Status Overview</h3>
                <p class="text-text-secondary-light dark:text-text-secondary-dark">No pod status metrics available</p>
            `;
            return;
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Pod Status Overview';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">Node</th>
                    <th class="p-4 font-semibold">Phase</th>
                    <th class="p-4 font-semibold">Status</th>
                    <th class="p-4 font-semibold">Running Containers</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.podStatusMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.nodeName}</td>
                <td class="p-4">${metric.podPhase}</td>
                <td class="p-4">
                    <span class="px-2 py-1 rounded-full text-xs font-medium
                        ${metric.derivedStatus === 'Running' ? 'bg-success/20 text-success dark:bg-success-dark/20 dark:text-success-dark' :
                          metric.derivedStatus === 'Pending' ? 'bg-warning/20 text-warning dark:bg-warning-dark/20 dark:text-warning-dark' :
                          'bg-danger/20 text-danger dark:bg-danger-dark/20 dark:text-danger-dark'}">
                        ${metric.derivedStatus}
                    </span>
                </td>
                <td class="p-4">${metric.runningContainers}/${metric.runningContainers + metric.nonRunningContainers}</td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        statusContainer.innerHTML = '';
        statusContainer.appendChild(title);
        statusContainer.appendChild(table);
    }

    // Update top memory pods in performance tab
    updateTopMemoryPodsInTab() {
        const tabContainer = document.getElementById('system-tab');
        if (!tabContainer || !this.podMetrics || !this.podMetrics.topPodMemoryMetrics) return;

        const memoryContainer = tabContainer.querySelector('.top-memory-pods');
        if (!memoryContainer) return;

        if (this.podMetrics.topPodMemoryMetrics.length === 0) {
            memoryContainer.innerHTML = `
                <h3 class="text-xl font-semibold mb-4">Top Memory Usage by Pod</h3>
                <p class="text-text-secondary-light dark:text-text-secondary-dark">No memory metrics available</p>
            `;
            return;
        }

        const title = document.createElement('h3');
        title.className = 'text-xl font-semibold mb-4';
        title.textContent = 'Top Memory Usage by Pod';

        const table = document.createElement('div');
        table.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const tableContent = document.createElement('table');
        tableContent.className = 'w-full text-left text-sm';

        tableContent.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Node</th>
                    <th class="p-4 font-semibold">Pod Name</th>
                    <th class="p-4 font-semibold">Memory Usage</th>
                    <th class="p-4 font-semibold">Last Update</th>
                </tr>
            </thead>
            <tbody class="divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = tableContent.querySelector('tbody');

        this.podMetrics.topPodMemoryMetrics.forEach(metric => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${metric.nodeIp}</td>
                <td class="p-4">${metric.podName}</td>
                <td class="p-4">${metric.memoryPct.toFixed(1)}%</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(metric.timestamp).toLocaleTimeString()}
                </td>
            `;
            tbody.appendChild(row);
        });

        table.appendChild(tableContent);

        // Clear existing content and add new
        memoryContainer.innerHTML = '';
        memoryContainer.appendChild(title);
        memoryContainer.appendChild(table);
    }

    // Start automatic updates
    startAutoUpdate() {
        setInterval(() => {
            this.fetchSystemHealth();
            this.fetchPodMetrics();
        }, this.updateInterval);
    }

    // Manual refresh
    async refresh() {
        await this.fetchSystemHealth();
        await this.fetchPodMetrics();
    }

    // Get current health data
    getHealthData() {
        return {
            health: this.healthData,
            pods: this.podMetrics,
            lastUpdate: this.lastUpdate
        };
    }
}

// Export for use in other modules
window.SystemHealthManager = SystemHealthManager;
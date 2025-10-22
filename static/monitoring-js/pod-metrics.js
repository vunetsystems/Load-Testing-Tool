// Pod Metrics JavaScript Module

class PodMetricsManager {
    constructor() {
        this.podMetrics = null;
        this.kubernetesPods = null;
        this.lastUpdate = null;
        this.updateInterval = 30000; // 30 seconds
        this.isUpdating = false;
    }

    // Initialize pod metrics monitoring
    async initialize() {
        console.log('PodMetricsManager: Initializing pod metrics monitoring...');
        try {
            await this.fetchPodMetrics();
            await this.fetchKubernetesPods();
            console.log('PodMetricsManager: Initialization completed successfully');
        } catch (error) {
            console.error('PodMetricsManager: Error during initialization:', error);
        }
        this.startAutoUpdate();
    }

    // Fetch pod metrics from API
    async fetchPodMetrics() {
        if (this.isUpdating) {
            console.log('PodMetricsManager: Already updating, skipping fetch');
            return;
        }

        console.log('PodMetricsManager: Starting fetchPodMetrics...');
        this.isUpdating = true;
        try {
            const response = await fetch('/api/clickhouse/pod-metrics');
            console.log('PodMetricsManager: API response status:', response.status);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('PodMetricsManager: API response data:', result);

            if (result.success && result.data) {
                this.podMetrics = result.data;
                console.log('PodMetricsManager: Pod metrics updated:', this.podMetrics);
                this.updatePodMetricsDisplay();
                this.lastUpdate = new Date();
                console.log('PodMetricsManager: Display updated, lastUpdate:', this.lastUpdate);
            } else {
                console.error('PodMetricsManager: Failed to fetch pod metrics:', result.message);
            }
        } catch (error) {
            console.error('PodMetricsManager: Error fetching pod metrics:', error);
        } finally {
            this.isUpdating = false;
            console.log('PodMetricsManager: fetchPodMetrics completed');
        }
    }

    // Update pod metrics display
    updatePodMetricsDisplay() {
        console.log('PodMetricsManager: updatePodMetricsDisplay called');
        console.log('PodMetricsManager: Current podMetrics:', this.podMetrics);

        // Update ClickHouse pod metrics if available
        if (this.podMetrics) {
            // Update summary cards
            this.updatePodSummaryCards();

            // Update resource metrics table
            this.updatePodResourceMetrics();

            // Update status metrics table
            this.updatePodStatusMetrics();

            // Update top memory pods table
            this.updateTopMemoryPods();
        }

        // Update Kubernetes pods table
        this.updateKubernetesPodsTable();

        console.log('PodMetricsManager: Display update completed');
    }

    // Update pod summary cards
    updatePodSummaryCards() {
        console.log('PodMetricsManager: updatePodSummaryCards called');

        if (this.podMetrics.podStatusMetrics) {
            const totalPods = this.podMetrics.podStatusMetrics.length;
            const runningPods = this.podMetrics.podStatusMetrics.filter(pod => pod.derivedStatus === 'Running').length;
            const failedPods = this.podMetrics.podStatusMetrics.filter(pod => pod.derivedStatus !== 'Running').length;

            console.log('PodMetricsManager: Pod counts - Total:', totalPods, 'Running:', runningPods, 'Failed:', failedPods);

            // Update total pods
            const totalPodsElement = document.getElementById('pod-total-pods');
            if (totalPodsElement) {
                totalPodsElement.textContent = totalPods;
                console.log('PodMetricsManager: Updated total pods element');
            } else {
                console.log('PodMetricsManager: pod-total-pods element not found');
            }

            // Update running pods
            const runningPodsElement = document.getElementById('pod-running-pods');
            if (runningPodsElement) {
                runningPodsElement.textContent = runningPods;
                console.log('PodMetricsManager: Updated running pods element');
            } else {
                console.log('PodMetricsManager: pod-running-pods element not found');
            }

            // Update failed pods
            const failedPodsElement = document.getElementById('pod-failed-pods');
            if (failedPodsElement) {
                failedPodsElement.textContent = failedPods;
                console.log('PodMetricsManager: Updated failed pods element');
            } else {
                console.log('PodMetricsManager: pod-failed-pods element not found');
            }
        }

        // Update average CPU usage
        if (this.podMetrics.podResourceMetrics && this.podMetrics.podResourceMetrics.length > 0) {
            const avgCpu = this.podMetrics.podResourceMetrics.reduce((sum, pod) => sum + pod.cpuPercentage, 0) / this.podMetrics.podResourceMetrics.length;
            const avgCpuElement = document.getElementById('pod-avg-cpu');
            if (avgCpuElement) {
                avgCpuElement.textContent = avgCpu.toFixed(1) + '%';
                console.log('PodMetricsManager: Updated avg CPU element');
            } else {
                console.log('PodMetricsManager: pod-avg-cpu element not found');
            }
        }
    }

    // Update pod resource metrics table
    updatePodResourceMetrics() {
        console.log('PodMetricsManager: updatePodResourceMetrics called');
        const tbody = document.getElementById('pod-resource-body');
        if (!tbody) {
            console.log('PodMetricsManager: pod-resource-body element not found');
            return;
        }

        if (!this.podMetrics.podResourceMetrics) {
            console.log('PodMetricsManager: No podResourceMetrics available');
            return;
        }

        console.log('PodMetricsManager: Updating resource metrics table with', this.podMetrics.podResourceMetrics.length, 'items');
        tbody.innerHTML = '';

        if (this.podMetrics.podResourceMetrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="4" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No pod resource metrics available</td>';
            tbody.appendChild(row);
            console.log('PodMetricsManager: Added no data message to resource table');
            return;
        }

        this.podMetrics.podResourceMetrics.forEach((metric, index) => {
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
            console.log('PodMetricsManager: Added resource metric row', index + 1, 'for pod:', metric.podName);
        });
    }

    // Update pod status metrics table
    updatePodStatusMetrics() {
        const tbody = document.getElementById('pod-status-body');
        if (!tbody || !this.podMetrics.podStatusMetrics) return;

        tbody.innerHTML = '';

        if (this.podMetrics.podStatusMetrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="5" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No pod status metrics available</td>';
            tbody.appendChild(row);
            return;
        }

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
    }

    // Update top memory pods table
    updateTopMemoryPods() {
        const tbody = document.getElementById('top-memory-body');
        if (!tbody || !this.podMetrics.topPodMemoryMetrics) return;

        tbody.innerHTML = '';

        if (this.podMetrics.topPodMemoryMetrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="4" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No memory metrics available</td>';
            tbody.appendChild(row);
            return;
        }

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
    }

    // Fetch Kubernetes pods from API
    async fetchKubernetesPods() {
        console.log('PodMetricsManager: Starting fetchKubernetesPods...');
        try {
            const response = await fetch('/api/kubernetes/pods');
            console.log('PodMetricsManager: Kubernetes API response status:', response.status);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('PodMetricsManager: Kubernetes API response data:', result);

            if (result.success && result.data) {
                this.kubernetesPods = result.data;
                console.log('PodMetricsManager: Kubernetes pods updated:', this.kubernetesPods);
                this.updatePodMetricsDisplay();
                this.lastUpdate = new Date();
                console.log('PodMetricsManager: Kubernetes display updated, lastUpdate:', this.lastUpdate);
            } else {
                console.error('PodMetricsManager: Failed to fetch Kubernetes pods:', result.message);
            }
        } catch (error) {
            console.error('PodMetricsManager: Error fetching Kubernetes pods:', error);
        }
        console.log('PodMetricsManager: fetchKubernetesPods completed');
    }

    // Update Kubernetes pods table
    updateKubernetesPodsTable() {
        console.log('PodMetricsManager: updateKubernetesPodsTable called');
        const tbody = document.getElementById('kubernetes-pods-body');
        if (!tbody) {
            console.log('PodMetricsManager: kubernetes-pods-body element not found');
            return;
        }

        if (!this.kubernetesPods) {
            console.log('PodMetricsManager: No kubernetesPods available');
            return;
        }

        console.log('PodMetricsManager: Updating Kubernetes pods table with', this.kubernetesPods.length, 'items');
        tbody.innerHTML = '';

        if (this.kubernetesPods.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="12" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No Kubernetes pods available</td>';
            tbody.appendChild(row);
            console.log('PodMetricsManager: Added no data message to Kubernetes table');
            return;
        }

        this.kubernetesPods.forEach((pod, index) => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4">${pod.namespace}</td>
                <td class="p-4">${pod.name}</td>
                <td class="p-4">${pod.node}</td>
                <td class="p-4">${pod.ready}</td>
                <td class="p-4">
                    <span class="px-2 py-1 rounded-full text-xs font-medium
                        ${pod.status === 'Running' ? 'bg-success/20 text-success dark:bg-success-dark/20 dark:text-success-dark' :
                          pod.status === 'Pending' ? 'bg-warning/20 text-warning dark:bg-warning-dark/20 dark:text-warning-dark' :
                          'bg-danger/20 text-danger dark:bg-danger-dark/20 dark:text-danger-dark'}">
                        ${pod.status}
                    </span>
                </td>
                <td class="p-4">${pod.cpu}</td>
                <td class="p-4">${pod.mem}</td>
                <td class="p-4">${pod.restarts}</td>
                <td class="p-4">${pod.last_restart}</td>
                <td class="p-4">${pod.ip}</td>
                <td class="p-4">${pod.qos}</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(pod.age).toLocaleString()}
                </td>
            `;
            tbody.appendChild(row);
            console.log('PodMetricsManager: Added Kubernetes pod row', index + 1, 'for pod:', pod.name);
        });
    }

    // Start automatic updates
    startAutoUpdate() {
        console.log('PodMetricsManager: Starting automatic updates every', this.updateInterval, 'ms');
        setInterval(() => {
            console.log('PodMetricsManager: Auto-update triggered');
            this.fetchPodMetrics();
            this.fetchKubernetesPods();
        }, this.updateInterval);
    }

    // Manual refresh
    async refresh() {
        console.log('PodMetricsManager: Manual refresh called');
        await this.fetchPodMetrics();
        await this.fetchKubernetesPods();
    }

    // Get current metrics
    getMetrics() {
        return {
            pods: this.podMetrics,
            kubernetesPods: this.kubernetesPods,
            lastUpdate: this.lastUpdate
        };
    }
}

// Export for use in other modules
window.PodMetricsManager = PodMetricsManager;
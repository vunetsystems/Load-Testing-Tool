// ClickHouse Metrics Module
class ClickHouseMetrics {
    constructor(manager) {
        this.manager = manager;
        this.allTopPodMemoryMetrics = []; // Store all top pod memory metrics for filtering
    }

    openClickHouseMetricsModal() {
        console.log('Opening ClickHouse metrics modal');
        const modal = this.manager.elements.clickHouseMetricsModal;

        if (!modal) {
            console.error('ClickHouse metrics modal not found!');
            return;
        }

        modal.classList.remove('hidden');
        this.refreshClickHouseMetrics();
    }

    closeClickHouseMetricsModal() {
        console.log('Closing ClickHouse metrics modal');
        const modal = this.manager.elements.clickHouseMetricsModal;

        if (!modal) {
            console.error('ClickHouse metrics modal not found!');
            return;
        }

        modal.classList.add('hidden');
    }

    async refreshClickHouseMetrics() {
        const refreshBtn = this.manager.elements.refreshClickHouseMetricsBtn;
        if (refreshBtn) {
            refreshBtn.disabled = true;
            refreshBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Refreshing...</span>';
        }

        try {
            // First check ClickHouse health
            const healthResponse = await this.manager.callAPI('/api/clickhouse/health');
            if (healthResponse.success && healthResponse.data) {
                this.updateClickHouseStatus(healthResponse.data);
            }

            // Then get metrics
            const metricsResponse = await this.manager.callAPI('/api/clickhouse/metrics');
            console.log('ClickHouse metrics API response:', metricsResponse);
            if (metricsResponse.success && metricsResponse.data) {
                console.log('ClickHouse metrics data:', metricsResponse.data);
                this.displayClickHouseMetrics(metricsResponse.data);
                this.manager.showNotification('ClickHouse metrics refreshed successfully', 'success');
            } else {
                this.manager.showNotification('Failed to load ClickHouse metrics', 'error');
            }
        } catch (error) {
            console.error('Error refreshing ClickHouse metrics:', error);
            this.manager.showNotification('Failed to refresh ClickHouse metrics: ' + error.message, 'error');
            this.updateClickHouseStatus({ status: 'error', error: error.message });
        } finally {
            if (refreshBtn) {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<span class="material-symbols-outlined">refresh</span><span>Refresh Metrics</span>';
            }
        }
    }

    updateClickHouseStatus(healthData) {
        const statusElement = this.manager.elements.clickHouseStatus;
        const lastUpdateElement = this.manager.elements.clickHouseLastUpdate;

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

        // Store all top pod memory metrics for filtering
        this.allTopPodMemoryMetrics = metrics.topPodMemoryMetrics || [];

        // Populate node filter dropdown
        this.populateNodeFilterDropdown();

        // Display Pod Metrics first
        this.displayPodResourceMetrics(metrics.podResourceMetrics || []);
        this.displayPodStatusMetrics(metrics.podStatusMetrics || []);
        this.displayTopPodMemoryMetrics(this.allTopPodMemoryMetrics);

        // Display Kafka Topic Metrics
        this.displayKafkaTopicMetrics(metrics.kafkaTopicMetrics || []);

        // Display System Metrics
        this.displaySystemMetrics(metrics.systemMetrics || []);

        // Display Database Metrics
        this.displayDatabaseMetrics(metrics.databaseMetrics || []);

        // Display Container Metrics
        this.displayContainerMetrics(metrics.containerMetrics || []);

        // Update last update time
        if (this.manager.elements.clickHouseLastUpdate && metrics.lastUpdated) {
            const timestamp = new Date(metrics.lastUpdated).toLocaleString();
            this.manager.elements.clickHouseLastUpdate.textContent = `Last updated: ${timestamp}`;
        }
    }

    displayPodResourceMetrics(metrics) {
        console.log('Displaying pod resource metrics:', metrics);
        const tbody = this.manager.elements.podResourceMetricsTable;
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
                const cpuPercentage = typeof metric.cpuPercentage === 'number' ? (metric.cpuPercentage * 100).toFixed(2) : 'N/A';
                const memoryPercentage = typeof metric.memoryPercentage === 'number' ? (metric.memoryPercentage * 100).toFixed(2) : 'N/A';
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
        const tbody = this.manager.elements.podStatusMetricsTable;
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
        const tbody = this.manager.elements.kafkaMetricsTable?.querySelector('tbody');
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
        const tbody = this.manager.elements.systemMetricsTable?.querySelector('tbody');
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
        const tbody = this.manager.elements.databaseMetricsTable?.querySelector('tbody');
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
        const tbody = this.manager.elements.containerMetricsTable?.querySelector('tbody');
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

    displayTopPodMemoryMetrics(metrics) {
        console.log('Displaying top pod memory metrics:', metrics);
        const tbody = this.manager.elements.topPodMemoryMetricsTable;
        if (!tbody) {
            console.error('Top pod memory metrics table not found in elements');
            return;
        }

        tbody.innerHTML = '';

        if (!metrics || !Array.isArray(metrics) || metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="4" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No top pod memory metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        metrics.forEach(metric => {
            try {
                const row = document.createElement('tr');
                row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

                // Handle potential missing or invalid values
                const timestamp = metric.timestamp ? new Date(metric.timestamp).toLocaleString() : 'N/A';
                const nodeIP = metric.nodeIp || 'N/A';
                const podName = metric.podName || 'N/A';
                const memoryPct = typeof metric.memoryPct === 'number' ? metric.memoryPct.toFixed(2) : 'N/A';

                row.innerHTML = `
                    <td class="p-3">${timestamp}</td>
                    <td class="p-3">${nodeIP}</td>
                    <td class="p-3">${podName}</td>
                    <td class="p-3 text-right">${memoryPct}${typeof metric.memoryPct === 'number' ? '%' : ''}</td>
                `;
                tbody.appendChild(row);
            } catch (error) {
                console.error('Error processing top pod memory metric:', error);
            }
        });
    }

    displayKafkaTopicMetrics(metrics) {
        console.log('Displaying Kafka topic metrics:', metrics);
        const tbody = this.manager.elements.kafkaTopicMetricsTable;
        if (!tbody) {
            console.error('Kafka topic metrics table not found in elements');
            return;
        }

        tbody.innerHTML = '';

        if (!metrics || !Array.isArray(metrics) || metrics.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="3" class="p-3 text-center text-text-secondary-light dark:text-text-secondary-dark">No Kafka topic metrics available</td>';
            tbody.appendChild(row);
            return;
        }

        // Group metrics by topic and get the latest for each topic
        const latestMetricsByTopic = {};
        metrics.forEach(metric => {
            if (metric.topic && typeof metric.oneMinuteRate === 'number') {
                if (!latestMetricsByTopic[metric.topic] ||
                    metric.timestamp > latestMetricsByTopic[metric.topic].timestamp) {
                    latestMetricsByTopic[metric.topic] = metric;
                }
            }
        });

        // Convert to array and sort by OneMinuteRate (highest first)
        const sortedMetrics = Object.values(latestMetricsByTopic)
            .sort((a, b) => b.oneMinuteRate - a.oneMinuteRate);

        sortedMetrics.forEach(metric => {
            try {
                const row = document.createElement('tr');
                row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

                // Handle potential missing or invalid values
                const timestamp = metric.timestamp ? new Date(metric.timestamp).toLocaleString() : 'N/A';
                const topic = metric.topic || 'N/A';
                const oneMinuteRate = typeof metric.oneMinuteRate === 'number' ? metric.oneMinuteRate.toFixed(2) : 'N/A';

                row.innerHTML = `
                    <td class="p-3">${timestamp}</td>
                    <td class="p-3">${topic}</td>
                    <td class="p-3 text-right">${oneMinuteRate}</td>
                `;
                tbody.appendChild(row);
            } catch (error) {
                console.error('Error processing Kafka topic metric:', error);
            }
        });
    }

    populateNodeFilterDropdown() {
        const nodeFilterSelect = this.manager.elements.nodeFilterSelect;
        if (!nodeFilterSelect) return;

        // Clear existing options except "All Nodes"
        while (nodeFilterSelect.children.length > 1) {
            nodeFilterSelect.removeChild(nodeFilterSelect.lastChild);
        }

        // Extract unique node IPs from the top pod memory metrics
        const nodeIPs = new Set();
        this.allTopPodMemoryMetrics.forEach(metric => {
            if (metric.nodeIp) {
                nodeIPs.add(metric.nodeIp);
            }
        });

        // Add node options
        Array.from(nodeIPs).sort().forEach(nodeIP => {
            const option = document.createElement('option');
            option.value = nodeIP;
            option.textContent = nodeIP;
            nodeFilterSelect.appendChild(option);
        });

        console.log(`Populated node filter dropdown with ${nodeIPs.size} nodes`);
    }

    filterTopPodMemoryMetrics() {
        const selectedNode = this.manager.elements.nodeFilterSelect.value;

        let filteredMetrics;
        if (!selectedNode) {
            // Show global top 5 pods across all nodes
            filteredMetrics = this.getGlobalTop5Pods();
        } else {
            // Filter by selected node (show top 5 for that specific node)
            filteredMetrics = this.allTopPodMemoryMetrics.filter(metric => metric.nodeIp === selectedNode);
        }

        this.displayTopPodMemoryMetrics(filteredMetrics);
    }

    getGlobalTop5Pods() {
        // Sort all pods globally by memory utilization (highest first) and take top 5
        return this.allTopPodMemoryMetrics
            .sort((a, b) => (b.memoryPct || 0) - (a.memoryPct || 0))
            .slice(0, 5);
    }
}
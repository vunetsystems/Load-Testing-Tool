// Kafka Metrics JavaScript Module

class KafkaMetricsManager {
    constructor() {
        this.kafkaData = null;
        this.lastUpdate = null;
        this.updateInterval = 10000; // 10 seconds
        this.isUpdating = false;
    }

    // Initialize Kafka metrics
    async initialize() {
        console.log('Initializing Kafka metrics...');
        await this.fetchKafkaMetrics();
        this.startAutoUpdate();
    }

    // Fetch Kafka topic metrics from API
    async fetchKafkaMetrics() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            const response = await fetch('http://164.52.213.158:8086/api/clickhouse/kafka-topics');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.kafkaData = result.data;
                this.lastUpdate = new Date();
                console.log('Kafka metrics updated:', this.kafkaData);
                this.updateKafkaDisplay();
            } else {
                console.error('Failed to fetch Kafka metrics:', result.message);
            }
        } catch (error) {
            console.error('Error fetching Kafka metrics:', error);
        } finally {
            this.isUpdating = false;
        }
    }

    // Update Kafka metrics display
    updateKafkaDisplay() {
        if (!this.kafkaData) return;

        // Update Kafka topics section
        this.updateKafkaTopicsSection();

        // Update performance metrics
        this.updatePerformanceMetrics();

        // Update summary cards
        this.updateSummaryCards();

        // Update charts if they exist
        this.updateCharts();
    }

    // Update Kafka topics section
    updateKafkaTopicsSection() {
        const performanceSection = document.getElementById('performance-section');
        if (!performanceSection) return;

        // Group metrics by topic
        const topicMetrics = {};
        this.kafkaData.forEach(metric => {
            if (!topicMetrics[metric.topic]) {
                topicMetrics[metric.topic] = [];
            }
            topicMetrics[metric.topic].push(metric);
        });

        // Create topics overview
        const topicsOverview = document.createElement('div');
        topicsOverview.className = 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8';

        Object.entries(topicMetrics).forEach(([topic, metrics]) => {
            const latestMetric = metrics.reduce((latest, current) =>
                new Date(current.timestamp) > new Date(latest.timestamp) ? current : latest
            );

            const avgRate = metrics.reduce((sum, m) => sum + m.oneMinuteRate, 0) / metrics.length;

            const topicCard = document.createElement('div');
            topicCard.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark p-6 shadow-md';

            topicCard.innerHTML = `
                <div class="flex items-center justify-between mb-4">
                    <h3 class="text-lg font-semibold">${topic}</h3>
                    <span class="material-symbols-outlined text-primary dark:text-primary-dark">topic</span>
                </div>
                <div class="space-y-3">
                    <div class="flex justify-between items-center">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Messages/sec</span>
                        <span class="font-semibold">${avgRate.toFixed(2)}</span>
                    </div>
                    <div class="flex justify-between items-center">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Latest Rate</span>
                        <span class="font-semibold">${latestMetric.oneMinuteRate.toFixed(2)}</span>
                    </div>
                    <div class="flex justify-between items-center">
                        <span class="text-sm text-text-secondary-light dark:text-text-secondary-dark">Data Points</span>
                        <span class="font-semibold">${metrics.length}</span>
                    </div>
                </div>
            `;

            topicsOverview.appendChild(topicCard);
        });

        // Clear existing content and add new content
        const existingOverview = performanceSection.querySelector('.kafka-topics-overview');
        if (existingOverview) {
            existingOverview.remove();
        }

        topicsOverview.className = 'kafka-topics-overview grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mb-8';
        performanceSection.appendChild(topicsOverview);

        // Create detailed metrics table
        this.createKafkaMetricsTable(topicMetrics);
    }

    // Create detailed Kafka metrics table
    createKafkaMetricsTable(topicMetrics) {
        const performanceSection = document.getElementById('performance-section');
        if (!performanceSection) return;

        const tableContainer = document.createElement('div');
        tableContainer.className = 'rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';

        const table = document.createElement('table');
        table.className = 'w-full text-left text-sm';

        table.innerHTML = `
            <thead class="bg-subtle-light/50 dark:bg-subtle-dark/50">
                <tr>
                    <th class="p-4 font-semibold">Topic</th>
                    <th class="p-4 font-semibold">Latest Rate</th>
                    <th class="p-4 font-semibold">Avg Rate</th>
                    <th class="p-4 font-semibold">Peak Rate</th>
                    <th class="p-4 font-semibold">Data Points</th>
                    <th class="p-4 font-semibold">Last Update</th>
                </tr>
            </thead>
            <tbody class="kafka-metrics-body divide-y divide-subtle-light dark:divide-subtle-dark">
            </tbody>
        `;

        const tbody = table.querySelector('.kafka-metrics-body');

        Object.entries(topicMetrics).forEach(([topic, metrics]) => {
            const latestMetric = metrics.reduce((latest, current) =>
                new Date(current.timestamp) > new Date(latest.timestamp) ? current : latest
            );

            const avgRate = metrics.reduce((sum, m) => sum + m.oneMinuteRate, 0) / metrics.length;
            const peakRate = Math.max(...metrics.map(m => m.oneMinuteRate));

            const row = document.createElement('tr');
            row.innerHTML = `
                <td class="p-4 font-medium">${topic}</td>
                <td class="p-4">${latestMetric.oneMinuteRate.toFixed(2)}</td>
                <td class="p-4">${avgRate.toFixed(2)}</td>
                <td class="p-4">${peakRate.toFixed(2)}</td>
                <td class="p-4">${metrics.length}</td>
                <td class="p-4 text-text-secondary-light dark:text-text-secondary-dark">
                    ${new Date(latestMetric.timestamp).toLocaleTimeString()}
                </td>
            `;

            tbody.appendChild(row);
        });

        tableContainer.appendChild(table);

        // Remove existing table and add new one
        const existingTable = performanceSection.querySelector('.kafka-metrics-table');
        if (existingTable) {
            existingTable.remove();
        }

        tableContainer.className = 'kafka-metrics-table rounded-xl border border-subtle-light dark:border-subtle-dark bg-surface-light dark:bg-surface-dark shadow-md overflow-hidden';
        performanceSection.appendChild(tableContainer);
    }

    // Update performance metrics
    updatePerformanceMetrics() {
        if (!this.kafkaData) return;

        // Calculate total message rate across all topics
        const totalRate = this.kafkaData.reduce((sum, metric) => sum + metric.oneMinuteRate, 0);

        // Update total EPS in overview section
        const totalEpsElement = document.querySelector('[data-stat="total-eps"]') ||
                               document.querySelector('.grid > div:nth-child(1) p:nth-child(2)');
        if (totalEpsElement) {
            totalEpsElement.textContent = Math.round(totalRate).toLocaleString();
        }
    }

    // Update summary cards in Kafka tab
    updateSummaryCards() {
        if (!this.kafkaData) return;

        // Group metrics by topic
        const topicMetrics = {};
        this.kafkaData.forEach(metric => {
            if (!topicMetrics[metric.topic]) {
                topicMetrics[metric.topic] = [];
            }
            topicMetrics[metric.topic].push(metric);
        });

        const topicCount = Object.keys(topicMetrics).length;
        const totalRate = this.kafkaData.reduce((sum, metric) => sum + metric.oneMinuteRate, 0);
        const avgRate = topicCount > 0 ? totalRate / topicCount : 0;

        // Update total topics
        const totalTopicsElement = document.getElementById('kafka-total-topics');
        if (totalTopicsElement) {
            totalTopicsElement.textContent = topicCount;
        }

        // Update total rate
        const totalRateElement = document.getElementById('kafka-total-rate');
        if (totalRateElement) {
            totalRateElement.textContent = totalRate.toFixed(2);
        }

        // Update average latency (placeholder - would need actual latency data)
        const avgLatencyElement = document.getElementById('kafka-avg-latency');
        if (avgLatencyElement) {
            avgLatencyElement.textContent = 'N/A';
        }

        // Update error rate (placeholder - would need actual error data)
        const errorRateElement = document.getElementById('kafka-error-rate');
        if (errorRateElement) {
            errorRateElement.textContent = '0.00%';
        }
    }

    // Update charts (placeholder for future chart implementation)
    updateCharts() {
        // This would update Chart.js or other charting libraries for Kafka metrics
        console.log('Kafka charts updated');
    }

    // Start automatic updates
    startAutoUpdate() {
        setInterval(() => {
            this.fetchKafkaMetrics();
        }, this.updateInterval);
    }

    // Manual refresh
    async refresh() {
        await this.fetchKafkaMetrics();
    }

    // Get current metrics data
    getMetrics() {
        return {
            data: this.kafkaData,
            lastUpdate: this.lastUpdate,
            topicCount: this.kafkaData ? new Set(this.kafkaData.map(m => m.topic)).size : 0
        };
    }
}

// Export for use in other modules
window.KafkaMetricsManager = KafkaMetricsManager;
// Kafka Metrics JavaScript Module

class KafkaMetricsManager {
    constructor() {
        this.kafkaData = null;
        this.lastUpdate = null;
        this.updateInterval = 10000; // 10 seconds
        this.isUpdating = false;

        // ECharts instances
        this.charts = {
            trend: null,
            comparison: null,
            distribution: null
        };

        // Chart configuration
        this.chartConfig = {
            trend: { type: 'line' },
            comparison: { type: 'bar' },
            distribution: { type: 'pie' }
        };

        // Chart colors for dark/light themes
        this.chartColors = {
            light: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16'],
            dark: ['#60a5fa', '#f87171', '#4ade80', '#fbbf24', '#a78bfa', '#22d3ee', '#f472b6', '#bef264']
        };
    }

    // Initialize Kafka metrics
    async initialize() {
        console.log('Initializing Kafka metrics...');
        this.initializeCharts();
        this.attachChartEventListeners();

        // Add resize listener for responsive charts
        window.addEventListener('resize', this.handleResize.bind(this));

        await this.fetchKafkaMetrics();
        this.startAutoUpdate();
    }

    // Initialize ECharts instances
    initializeCharts() {
        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        // Initialize trend chart (line/area chart)
        const trendChartDom = document.getElementById('kafka-trend-chart');
        if (trendChartDom) {
            this.charts.trend = echarts.init(trendChartDom);
        }

        // Initialize comparison chart (bar/horizontal bar chart)
        const comparisonChartDom = document.getElementById('kafka-comparison-chart');
        if (comparisonChartDom) {
            this.charts.comparison = echarts.init(comparisonChartDom);
        }

        // Initialize distribution chart (pie/doughnut chart)
        const distributionChartDom = document.getElementById('kafka-distribution-chart');
        if (distributionChartDom) {
            this.charts.distribution = echarts.init(distributionChartDom);
        }

        console.log('ECharts initialized for Kafka metrics');
    }

    // Attach event listeners for chart type buttons
    attachChartEventListeners() {
        document.addEventListener('click', (event) => {
            if (event.target.classList.contains('kafka-chart-type-btn')) {
                const chartType = event.target.getAttribute('data-chart');
                const newType = event.target.getAttribute('data-type');

                this.changeChartType(chartType, newType);

                // Update button states
                const buttons = document.querySelectorAll(`[data-chart="${chartType}"]`);
                buttons.forEach(btn => {
                    if (btn.getAttribute('data-type') === newType) {
                        btn.className = 'kafka-chart-type-btn px-3 py-1 text-xs rounded-md bg-primary/20 text-primary dark:bg-primary-dark/20 dark:text-primary-dark';
                    } else {
                        btn.className = 'kafka-chart-type-btn px-3 py-1 text-xs rounded-md bg-subtle-light dark:bg-subtle-dark text-text-secondary-light dark:text-text-secondary-dark';
                    }
                });
            }
        });
    }

    // Change chart type
    changeChartType(chartName, type) {
        this.chartConfig[chartName].type = type;
        this.updateCharts();
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

    // Update charts with current data
    updateCharts() {
        if (!this.kafkaData || this.kafkaData.length === 0) {
            this.showEmptyCharts();
            return;
        }

        this.updateTrendChart();
        this.updateComparisonChart();
        this.updateDistributionChart();
        this.updateLegend();

        console.log('Kafka charts updated');
    }

    // Show empty state for charts when no data
    showEmptyCharts() {
        const emptyOption = {
            title: {
                text: 'No Data Available',
                left: 'center',
                top: 'center',
                textStyle: {
                    color: '#64748b',
                    fontSize: 14
                }
            }
        };

        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.setOption(emptyOption, true);
            }
        });
    }

    // Update trend chart (line/area chart for message rates over time)
    updateTrendChart() {
        if (!this.charts.trend) return;

        // Group data by topic and prepare time series
        const topicData = {};
        this.kafkaData.forEach(metric => {
            if (!topicData[metric.topic]) {
                topicData[metric.topic] = [];
            }
            topicData[metric.topic].push({
                timestamp: new Date(metric.timestamp).toLocaleTimeString(),
                rate: metric.oneMinuteRate
            });
        });

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        const series = Object.entries(topicData).map(([topic, data], index) => ({
            name: topic,
            type: this.chartConfig.trend.type === 'area' ? 'line' : 'line',
            areaStyle: this.chartConfig.trend.type === 'area' ? {} : undefined,
            data: data.map(d => d.rate),
            itemStyle: { color: colors[index % colors.length] },
            smooth: true
        }));

        const option = {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' }
            },
            legend: {
                data: Object.keys(topicData),
                bottom: 0
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                top: '10%',
                containLabel: true
            },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: this.kafkaData.filter((m, index, arr) =>
                    arr.findIndex(am => am.timestamp === m.timestamp) === index
                ).map(m => new Date(m.timestamp).toLocaleTimeString()).filter((time, index, arr) => arr.indexOf(time) === index)
            },
            yAxis: {
                type: 'value',
                name: 'Messages/sec',
                nameLocation: 'middle',
                nameGap: 30
            },
            series: series
        };

        this.charts.trend.setOption(option, true);
    }

    // Update comparison chart (bar/horizontal bar chart)
    updateComparisonChart() {
        if (!this.charts.comparison) return;

        // Calculate average rates by topic
        const topicAverages = {};
        this.kafkaData.forEach(metric => {
            if (!topicAverages[metric.topic]) {
                topicAverages[metric.topic] = { sum: 0, count: 0 };
            }
            topicAverages[metric.topic].sum += metric.oneMinuteRate;
            topicAverages[metric.topic].count += 1;
        });

        const topics = Object.keys(topicAverages);
        const rates = topics.map(topic =>
            (topicAverages[topic].sum / topicAverages[topic].count).toFixed(2)
        );

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        const option = {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'shadow' }
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '3%',
                top: '3%',
                containLabel: true
            },
            xAxis: this.chartConfig.comparison.type === 'horizontalBar' ? {
                type: 'value',
                name: 'Messages/sec',
                nameLocation: 'middle',
                nameGap: 30
            } : {
                type: 'category',
                data: topics,
                axisLabel: { rotate: 45 }
            },
            yAxis: this.chartConfig.comparison.type === 'horizontalBar' ? {
                type: 'category',
                data: topics
            } : {
                type: 'value',
                name: 'Messages/sec',
                nameLocation: 'middle',
                nameGap: 30
            },
            series: [{
                name: 'Average Rate',
                type: this.chartConfig.comparison.type === 'horizontalBar' ? 'bar' : 'bar',
                data: rates,
                itemStyle: {
                    color: (params) => colors[params.dataIndex % colors.length]
                }
            }]
        };

        this.charts.comparison.setOption(option, true);
    }

    // Update distribution chart (pie/doughnut chart)
    updateDistributionChart() {
        if (!this.charts.distribution) return;

        // Calculate total rates by topic
        const topicTotals = {};
        this.kafkaData.forEach(metric => {
            if (!topicTotals[metric.topic]) {
                topicTotals[metric.topic] = 0;
            }
            topicTotals[metric.topic] += metric.oneMinuteRate;
        });

        const topics = Object.keys(topicTotals);
        const totals = topics.map(topic => topicTotals[topic]);

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        const option = {
            tooltip: {
                trigger: 'item',
                formatter: '{a} <br/>{b}: {c} ({d}%)'
            },
            legend: {
                type: 'scroll',
                bottom: 0,
                data: topics
            },
            series: [{
                name: 'Topic Distribution',
                type: 'pie',
                radius: this.chartConfig.distribution.type === 'doughnut' ? ['40%', '70%'] : ['0%', '70%'],
                center: ['50%', '45%'],
                avoidLabelOverlap: false,
                itemStyle: {
                    color: (params) => colors[params.dataIndex % colors.length]
                },
                label: {
                    show: false,
                    position: 'center'
                },
                emphasis: {
                    label: {
                        show: true,
                        fontSize: '18',
                        fontWeight: 'bold'
                    }
                },
                labelLine: {
                    show: false
                },
                data: topics.map((topic, index) => ({
                    value: totals[index],
                    name: topic
                }))
            }]
        };

        this.charts.distribution.setOption(option, true);
    }

    // Update legend for distribution chart
    updateLegend() {
        const legendContainer = document.querySelector('.kafka-topics-legend');
        if (!legendContainer) return;

        // Calculate total rates by topic
        const topicTotals = {};
        this.kafkaData.forEach(metric => {
            if (!topicTotals[metric.topic]) {
                topicTotals[metric.topic] = 0;
            }
            topicTotals[metric.topic] += metric.oneMinuteRate;
        });

        const totalMessages = Object.values(topicTotals).reduce((sum, val) => sum + val, 0);

        legendContainer.innerHTML = Object.entries(topicTotals)
            .sort(([,a], [,b]) => b - a)
            .map(([topic, total], index) => {
                const percentage = ((total / totalMessages) * 100).toFixed(1);
                const isDark = document.documentElement.classList.contains('dark');
                const colors = isDark ? this.chartColors.dark : this.chartColors.light;

                return `
                    <div class="flex items-center gap-2 mb-2">
                        <div class="w-3 h-3 rounded-full" style="background-color: ${colors[index % colors.length]}"></div>
                        <span class="text-sm text-text-light dark:text-text-dark flex-1">${topic}</span>
                        <span class="text-sm font-semibold">${total.toFixed(2)}</span>
                        <span class="text-xs text-text-secondary-light dark:text-text-secondary-dark">(${percentage}%)</span>
                    </div>
                `;
            }).join('');
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

    // Handle window resize for responsive charts
    handleResize() {
        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.resize();
            }
        });
    }

    // Cleanup charts
    destroy() {
        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.dispose();
            }
        });
        this.charts = {};

        // Remove resize listener
        window.removeEventListener('resize', this.handleResize.bind(this));
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
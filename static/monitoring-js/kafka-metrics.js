// Kafka Metrics JavaScript Module

class KafkaMetricsManager {
    constructor() {
        this.kafkaData = null;
        this.lastUpdate = null;
        this.updateInterval = 10000; // 10 seconds
        this.isUpdating = false;

        // In-memory data storage for last 5 minutes
        this.dataBuffer = new Map(); // topic -> array of data points
        this.maxDataAge = 5 * 60 * 1000; // 5 minutes in milliseconds
        this.maxDataPoints = 100; // Maximum points per topic to prevent memory issues

        // ECharts instances
        this.charts = {
            inputTrend: null,
            outputTrend: null,
            comparison: null,
            distribution: null
        };

        // Chart configuration
        this.chartConfig = {
            inputTrend: { type: 'line' },
            outputTrend: { type: 'line' },
            comparison: { type: 'bar' },
            distribution: { type: 'pie' }
        };

        // Chart colors for dark/light themes
        this.chartColors = {
            light: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#06b6d4', '#ec4899', '#84cc16'],
            dark: ['#60a5fa', '#f87171', '#4ade80', '#fbbf24', '#a78bfa', '#22d3ee', '#f472b6', '#bef264']
        };

        // Loading states
        this.isInitialized = false;
        this.chartsInitialized = false;

        // Test run filtering
        this.testRuns = [];
        this.selectedTestRun = null;
        this.isTestFiltered = false;

        // Auto-update tracking
        this.updateIntervalId = null;
    }

    // Initialize Kafka metrics
    async initialize() {
        console.log('Initializing Kafka metrics...');
        this.isInitialized = true;

        // Load test runs for dropdown
        await this.fetchTestRuns();

        // Don't initialize charts immediately - wait for section to be visible
        this.attachChartEventListeners();
        this.attachTestRunEventListeners();

        // Add resize listener for responsive charts
        window.addEventListener('resize', this.handleResize.bind(this));

        await this.fetchKafkaMetrics();
        this.startAutoUpdate();
    }

    // Initialize ECharts instances - only when DOM elements are available
    initializeCharts() {
        if (this.chartsInitialized) return;

        try {
            // Get theme colors
            const isDark = document.documentElement.classList.contains('dark');
            const colors = isDark ? this.chartColors.dark : this.chartColors.light;

            // Initialize input trend chart (line/area chart)
            const inputTrendChartDom = document.getElementById('kafka-input-trend-chart');
            if (inputTrendChartDom) {
                this.charts.inputTrend = echarts.init(inputTrendChartDom);
                console.log('Input trend chart initialized');
            } else {
                console.warn('Input trend chart DOM element not found');
            }

            // Initialize output trend chart (line/area chart)
            const outputTrendChartDom = document.getElementById('kafka-output-trend-chart');
            if (outputTrendChartDom) {
                this.charts.outputTrend = echarts.init(outputTrendChartDom);
                console.log('Output trend chart initialized');
            } else {
                console.warn('Output trend chart DOM element not found');
            }

            // Initialize comparison chart (bar/horizontal bar chart)
            const comparisonChartDom = document.getElementById('kafka-comparison-chart');
            if (comparisonChartDom) {
                this.charts.comparison = echarts.init(comparisonChartDom);
                console.log('Comparison chart initialized');
            } else {
                console.warn('Comparison chart DOM element not found');
            }

            // Initialize distribution chart (pie/doughnut chart)
            const distributionChartDom = document.getElementById('kafka-distribution-chart');
            if (distributionChartDom) {
                this.charts.distribution = echarts.init(distributionChartDom);
                console.log('Distribution chart initialized');
            } else {
                console.warn('Distribution chart DOM element not found');
            }

            this.chartsInitialized = true;
            console.log('ECharts initialized for Kafka metrics');

        } catch (error) {
            console.error('Error initializing charts:', error);
            this.showChartError('Failed to initialize charts. Please refresh the page.');
        }
    }

    // Show error message in chart containers
    showChartError(message) {
        const errorOption = {
            title: {
                text: 'Error',
                left: 'center',
                top: 'center',
                textStyle: {
                    color: '#ef4444',
                    fontSize: 14
                }
            },
            graphic: {
                type: 'text',
                left: 'center',
                top: 'middle',
                style: {
                    text: message,
                    fontSize: 12,
                    fill: '#64748b'
                }
            }
        };

        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.setOption(errorOption, true);
            }
        });
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

    // Attach event listeners for test run dropdown
    attachTestRunEventListeners() {
        const dropdown = document.getElementById('kafka-test-run-dropdown');
        const clearBtn = document.getElementById('kafka-clear-test-filter');

        if (dropdown) {
            dropdown.addEventListener('change', (event) => {
                this.onTestRunChange(event.target.value);
            });
        }

        if (clearBtn) {
            clearBtn.addEventListener('click', () => {
                this.clearTestFilter();
            });
        }
    }

    // Fetch test runs for dropdown
    async fetchTestRuns() {
        try {
            const response = await fetch('/api/test-runs/dropdown');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.testRuns = result.data;
                this.populateTestRunDropdown();
                console.log('Test runs loaded:', this.testRuns.length);
            } else {
                console.error('Failed to fetch test runs:', result.message);
            }
        } catch (error) {
            console.error('Error fetching test runs:', error);
        }
    }

    // Populate dropdown with test runs
    populateTestRunDropdown() {
        const dropdown = document.getElementById('kafka-test-run-dropdown');
        if (!dropdown) return;

        // Clear existing options except the first one
        dropdown.innerHTML = '<option value="">Select a test run...</option>';

        // Add test run options
        this.testRuns.forEach(testRun => {
            const option = document.createElement('option');
            option.value = testRun.test_id;
            option.textContent = `${testRun.test_id} (${testRun.duration})`;
            dropdown.appendChild(option);
        });
    }

    // Handle test run selection
    onTestRunChange(testId) {
        console.log('onTestRunChange called with testId:', testId);
        if (!testId) {
            this.clearTestFilter();
            return;
        }

        const selectedTest = this.testRuns.find(test => test.test_id === testId);
        if (selectedTest) {
            this.selectedTestRun = selectedTest;
            this.isTestFiltered = true;

            // Clear real-time buffer to avoid mixing with historical data
            this.dataBuffer.clear();

            // Update UI
            this.updateSelectedTestInfo();

            // Refresh data with time filter
            console.log('Fetching Kafka metrics for test run:', selectedTest.test_id);
            this.fetchKafkaMetrics();

            console.log('Test run selected:', selectedTest);
        } else {
            console.error('Test run not found for ID:', testId);
        }
    }

    // Clear test filter
    clearTestFilter() {
        this.selectedTestRun = null;
        this.isTestFiltered = false;

        // Reset dropdown
        const dropdown = document.getElementById('kafka-test-run-dropdown');
        if (dropdown) {
            dropdown.value = '';
        }

        // Hide test info
        const infoDiv = document.getElementById('kafka-selected-test-info');
        if (infoDiv) {
            infoDiv.classList.add('hidden');
        }

        // Refresh data without filter and restart auto-update
        this.fetchKafkaMetrics();
        this.startAutoUpdate();

        console.log('Test filter cleared');
    }

    // Update selected test info display
    updateSelectedTestInfo() {
        const infoDiv = document.getElementById('kafka-selected-test-info');
        const nameSpan = document.getElementById('kafka-selected-test-name');
        const rangeSpan = document.getElementById('kafka-selected-test-range');
        const durationSpan = document.getElementById('kafka-selected-test-duration');

        if (!infoDiv || !this.selectedTestRun) return;

        nameSpan.textContent = this.selectedTestRun.test_name || 'Unnamed Test';
        rangeSpan.textContent = `${new Date(this.selectedTestRun.start_time).toLocaleString()} - ${new Date(this.selectedTestRun.end_time).toLocaleString()}`;
        durationSpan.textContent = this.selectedTestRun.duration;

        infoDiv.classList.remove('hidden');
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

        // Show loading state in charts if they're initialized
        if (this.chartsInitialized) {
            this.showLoadingCharts();
        }

        try {
            // Build API URL with optional time range and test_id parameters
            let apiUrl = '/api/clickhouse/kafka-topics';

            if (this.isTestFiltered && this.selectedTestRun) {
                const startTime = new Date(this.selectedTestRun.start_time).toISOString();
                const endTime = new Date(this.selectedTestRun.end_time).toISOString();
                apiUrl += `?start=${encodeURIComponent(startTime)}&end=${encodeURIComponent(endTime)}&test_id=${encodeURIComponent(this.selectedTestRun.test_id)}`;
            }

            const response = await fetch(apiUrl);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                // Store data in buffer for historical tracking
                this.storeDataInBuffer(result.data);

                this.kafkaData = result.data;
                this.lastUpdate = new Date();
                console.log('Kafka metrics updated:', this.kafkaData.length, 'records, isTestFiltered:', this.isTestFiltered);

                // Only update display if charts are initialized (section is visible)
                if (this.chartsInitialized) {
                    this.updateKafkaDisplay();
                }
            } else {
                console.error('Failed to fetch Kafka metrics:', result.message);
                this.showApiError('Failed to fetch Kafka metrics from server');
            }
        } catch (error) {
            console.error('Error fetching Kafka metrics:', error);
            this.showApiError('Network error while fetching Kafka metrics');
        } finally {
            this.isUpdating = false;
        }
    }

    // Store data in in-memory buffer for last 5 minutes
    storeDataInBuffer(newData) {
        const now = Date.now();

        newData.forEach(metric => {
            const topic = metric.topic;
            if (!this.dataBuffer.has(topic)) {
                this.dataBuffer.set(topic, []);
            }

            const topicData = this.dataBuffer.get(topic);
            const dataPoint = {
                timestamp: now,
                oneMinuteRate: metric.oneMinuteRate,
                count: metric.count,
                rawData: metric
            };

            topicData.push(dataPoint);

            // Keep only last 5 minutes of data
            const cutoffTime = now - this.maxDataAge;
            const filteredData = topicData.filter(point => point.timestamp >= cutoffTime);

            // Limit to maxDataPoints to prevent memory issues
            if (filteredData.length > this.maxDataPoints) {
                filteredData.splice(0, filteredData.length - this.maxDataPoints);
            }

            this.dataBuffer.set(topic, filteredData);
        });

        // Clean up old data periodically (every 100 data points)
        if (this.dataBuffer.size > 0 && Math.random() < 0.01) {
            this.cleanupOldData();
        }
    }

    // Clean up old data across all topics
    cleanupOldData() {
        const now = Date.now();
        const cutoffTime = now - this.maxDataAge;

        for (const [topic, data] of this.dataBuffer.entries()) {
            const filteredData = data.filter(point => point.timestamp >= cutoffTime);
            if (filteredData.length === 0) {
                this.dataBuffer.delete(topic);
            } else {
                this.dataBuffer.set(topic, filteredData);
            }
        }
    }

    // Show API error in UI
    showApiError(message) {
        // Show error in summary cards
        const errorElements = [
            document.getElementById('kafka-total-topics'),
            document.getElementById('kafka-total-rate'),
            document.getElementById('kafka-avg-latency'),
            document.getElementById('kafka-error-rate')
        ];

        errorElements.forEach(element => {
            if (element) {
                element.textContent = 'Error';
                element.style.color = '#ef4444';
            }
        });

        // Show error in charts if they're initialized
        if (this.chartsInitialized) {
            this.showChartError(message);
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
<div class="bg-white p-4 rounded-2xl shadow">
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
<thead class="bg-white">
  <tr>
    <th class="p-4 font-semibold">Topic</th>
    <th class="p-4 font-semibold">Latest Rate</th>
    <th class="p-4 font-semibold">Avg Rate</th>
    <th class="p-4 font-semibold">Peak Rate</th>
    <th class="p-4 font-semibold">Data Points</th>
    <th class="p-4 font-semibold">Last Update</th>
  </tr>
</thead>

<tbody class="kafka-metrics-body divide-y divide-subtle-light dark:divide-subtle-dark bg-white">
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
        if (!this.chartsInitialized) {
            console.log('Charts not initialized yet, skipping update');
            return;
        }

        if (!this.kafkaData || this.kafkaData.length === 0) {
            console.log('No data available, showing empty charts');
            this.showEmptyCharts();
            return;
        }

        try {
            console.log('Updating charts with data:', this.kafkaData.length, 'records, isTestFiltered:', this.isTestFiltered);
            this.updateInputTrendChart();
            this.updateOutputTrendChart();
            this.updateComparisonChart();
            this.updateDistributionChart();
            this.updateLegend();

            console.log('Kafka charts updated successfully');
        } catch (error) {
            console.error('Error updating charts:', error);
            this.showChartError('Error updating charts. Please refresh the page.');
        }
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
            },
            graphic: {
                type: 'text',
                left: 'center',
                top: 'middle',
                style: {
                    text: 'Waiting for Kafka metrics...',
                    fontSize: 12,
                    fill: '#64748b'
                }
            }
        };

        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.setOption(emptyOption, true);
            }
        });
    }

    // Show loading state for charts
    showLoadingCharts() {
        const loadingOption = {
            title: {
                text: 'Loading...',
                left: 'center',
                top: 'center',
                textStyle: {
                    color: '#64748b',
                    fontSize: 14
                }
            },
            graphic: {
                type: 'text',
                left: 'center',
                top: 'middle',
                style: {
                    text: 'Fetching Kafka metrics...',
                    fontSize: 12,
                    fill: '#64748b'
                }
            }
        };

        Object.values(this.charts).forEach(chart => {
            if (chart) {
                chart.setOption(loadingOption, true);
            }
        });
    }

    // Update input trend chart (line/area chart for input topic message rates over time)
    updateInputTrendChart() {
        if (!this.charts.inputTrend) return;

        let inputTopicData = {};

        if (this.isTestFiltered && this.selectedTestRun && this.kafkaData) {
            // Use API response data for filtered test view (historical data)
            this.kafkaData.forEach(metric => {
                if (metric.topic.includes('-input') || metric.topic === 'mssql-telegraf') {
                    if (!inputTopicData[metric.topic]) {
                        inputTopicData[metric.topic] = [];
                    }
                    inputTopicData[metric.topic].push({
                        timestamp: new Date(metric.timestamp).getTime(),
                        rate: metric.oneMinuteRate
                    });
                }
            });
        } else {
            // Use real-time buffer data for live monitoring
            const now = Date.now();

            // Get data from buffer for input topics only
            for (const [topic, dataPoints] of this.dataBuffer.entries()) {
                if (topic.includes('-input') || topic === 'mssql-telegraf') {
                    inputTopicData[topic] = dataPoints.map(point => ({
                        timestamp: point.timestamp,
                        rate: point.oneMinuteRate
                    }));
                }
            }

            // If no buffer data, fall back to current data
            if (Object.keys(inputTopicData).length === 0 && this.kafkaData) {
                this.kafkaData.forEach(metric => {
                    if (metric.topic.includes('-input') || metric.topic === 'mssql-telegraf') {
                        if (!inputTopicData[metric.topic]) {
                            inputTopicData[metric.topic] = [];
                        }
                        inputTopicData[metric.topic].push({
                            timestamp: new Date(metric.timestamp).getTime(),
                            rate: metric.oneMinuteRate
                        });
                    }
                });
            }
        }

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        // Prepare series data for input topics
        const series = Object.entries(inputTopicData).map(([topic, data], index) => {
            // Sort data by timestamp
            data.sort((a, b) => a.timestamp - b.timestamp);

            return {
                name: `📥 ${topic}`,
                type: 'line',
                areaStyle: this.chartConfig.inputTrend.type === 'area' ? {} : undefined,
                data: data.map(d => [d.timestamp, d.rate]),
                itemStyle: { color: colors[index % colors.length] },
                lineStyle: { width: 2 }, // Solid lines for input topics
                smooth: true,
                showSymbol: false // Hide symbols for better performance
            };
        });

        // Configure X-axis based on data mode
        let xAxisConfig = {
            type: 'time',
            boundaryGap: false,
            name: 'Time',
            nameLocation: 'middle',
            nameGap: 30
        };

        // For filtered test data, set proper time range and granularity
        if (this.isTestFiltered && this.selectedTestRun) {
            const testStart = new Date(this.selectedTestRun.start_time).getTime();
            const testEnd = new Date(this.selectedTestRun.end_time).getTime();

            xAxisConfig.min = testStart;
            xAxisConfig.max = testEnd;

            xAxisConfig.axisLabel = {
                formatter: (value) => {
                    const date = new Date(value);
                    const testDuration = testEnd - testStart;

                    // For short tests (< 1 hour), show HH:MM:SS
                    if (testDuration < 60 * 60 * 1000) {
                        return date.toLocaleTimeString('en-US', {
                            hour12: false,
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit'
                        });
                    }
                    // For longer tests, show HH:MM
                    return date.toLocaleTimeString('en-US', {
                        hour12: false,
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                }
            };
        }

        const option = {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
                formatter: (params) => {
                    const date = new Date(params[0].value[0]);
                    let result = date.toLocaleTimeString() + '<br/>';
                    params.forEach(param => {
                        const cleanName = param.seriesName.replace(/^📥\s*/, '');
                        result += `Input: ${cleanName}<br/>${param.value[1].toFixed(2)} msg/sec<br/>`;
                    });
                    return result;
                }
            },
            legend: {
                data: series.map(s => s.name),
                bottom: 0,
                type: 'scroll'
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                top: '10%',
                containLabel: true
            },
            xAxis: xAxisConfig,
            yAxis: {
                type: 'value',
                name: 'Total Messages',
                nameLocation: 'middle',
                nameGap: 30
            },
            series: series
        };

        this.charts.inputTrend.setOption(option, true);
    }

    // Update output trend chart (line/area chart for output topic message rates over time)
    updateOutputTrendChart() {
        if (!this.charts.outputTrend) return;

        let outputTopicData = {};

        if (this.isTestFiltered && this.selectedTestRun && this.kafkaData) {
            // Use API response data for filtered test view (historical data)
            this.kafkaData.forEach(metric => {
                if (!metric.topic.includes('-input') && metric.topic !== 'mssql-telegraf') {
                    if (!outputTopicData[metric.topic]) {
                        outputTopicData[metric.topic] = [];
                    }
                    outputTopicData[metric.topic].push({
                        timestamp: new Date(metric.timestamp).getTime(),
                        count: metric.count
                    });
                }
            });
        } else {
            // Use real-time buffer data for live monitoring
            const now = Date.now();

            // Get data from buffer for output topics only
            for (const [topic, dataPoints] of this.dataBuffer.entries()) {
                if (!topic.includes('-input') && topic !== 'mssql-telegraf') {
                    outputTopicData[topic] = dataPoints.map(point => ({
                        timestamp: point.timestamp,
                        count: point.rawData ? point.rawData.count : 0
                    }));
                }
            }

            // If no buffer data, fall back to current data
            if (Object.keys(outputTopicData).length === 0 && this.kafkaData) {
                this.kafkaData.forEach(metric => {
                    if (!metric.topic.includes('-input') && metric.topic !== 'mssql-telegraf') {
                        if (!outputTopicData[metric.topic]) {
                            outputTopicData[metric.topic] = [];
                        }
                        outputTopicData[metric.topic].push({
                            timestamp: new Date(metric.timestamp).getTime(),
                            count: metric.count
                        });
                    }
                });
            }
        }

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        // Prepare series data for output topics
        const series = Object.entries(outputTopicData).map(([topic, data], index) => {
            // Sort data by timestamp
            data.sort((a, b) => a.timestamp - b.timestamp);

            return {
                name: `📤 ${topic}`,
                type: 'line',
                areaStyle: this.chartConfig.outputTrend.type === 'area' ? {} : undefined,
                data: data.map(d => [d.timestamp, d.count]),
                itemStyle: { color: colors[index % colors.length] },
                lineStyle: { width: 1.5, type: 'dashed' }, // Dashed lines for output topics
                smooth: true,
                showSymbol: false // Hide symbols for better performance
            };
        });

        // Configure X-axis based on data mode
        let xAxisConfig = {
            type: 'time',
            boundaryGap: false,
            name: 'Time',
            nameLocation: 'middle',
            nameGap: 30
        };

        // For filtered test data, set proper time range and granularity
        if (this.isTestFiltered && this.selectedTestRun) {
            const testStart = new Date(this.selectedTestRun.start_time).getTime();
            const testEnd = new Date(this.selectedTestRun.end_time).getTime();

            xAxisConfig.min = testStart;
            xAxisConfig.max = testEnd;

            xAxisConfig.axisLabel = {
                formatter: (value) => {
                    const date = new Date(value);
                    const testDuration = testEnd - testStart;

                    // For short tests (< 1 hour), show HH:MM:SS
                    if (testDuration < 60 * 60 * 1000) {
                        return date.toLocaleTimeString('en-US', {
                            hour12: false,
                            hour: '2-digit',
                            minute: '2-digit',
                            second: '2-digit'
                        });
                    }
                    // For longer tests, show HH:MM
                    return date.toLocaleTimeString('en-US', {
                        hour12: false,
                        hour: '2-digit',
                        minute: '2-digit'
                    });
                }
            };
        }

        const option = {
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
                formatter: (params) => {
                    const date = new Date(params[0].value[0]);
                    let result = date.toLocaleTimeString() + '<br/>';
                    params.forEach(param => {
                        const cleanName = param.seriesName.replace(/^📤\s*/, '');
                        result += `Output: ${cleanName}<br/>${param.value[1].toLocaleString()} messages<br/>`;
                    });
                    return result;
                }
            },
            legend: {
                data: series.map(s => s.name),
                bottom: 0,
                type: 'scroll'
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                top: '10%',
                containLabel: true
            },
            xAxis: xAxisConfig,
            yAxis: {
                type: 'value',
                name: 'Messages/sec',
                nameLocation: 'middle',
                nameGap: 30
            },
            series: series
        };

        this.charts.outputTrend.setOption(option, true);
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
        // Clear any existing interval
        if (this.updateIntervalId) {
            clearInterval(this.updateIntervalId);
        }

        // Only start auto-update if not in test filter mode
        if (!this.isTestFiltered) {
            this.updateIntervalId = setInterval(() => {
                this.fetchKafkaMetrics();
            }, this.updateInterval);
        }
    }

    // Manual refresh
    async refresh() {
        await this.fetchKafkaMetrics();
    }

    // Initialize charts when section becomes visible
    onSectionVisible() {
        console.log('onSectionVisible called, chartsInitialized:', this.chartsInitialized, 'isInitialized:', this.isInitialized);
        if (!this.chartsInitialized && this.isInitialized) {
            console.log('Kafka metrics section became visible, initializing charts...');

            // Show loading state immediately
            setTimeout(() => {
                this.initializeCharts();

                // Update display with current data
                if (this.kafkaData) {
                    console.log('Updating display with existing data after chart initialization');
                    this.updateKafkaDisplay();
                } else {
                    // If no data yet, show loading state
                    console.log('No data available, showing loading state');
                    this.showLoadingCharts();
                }
            }, 100); // Small delay to ensure DOM is ready
        } else if (this.chartsInitialized && this.kafkaData) {
            console.log('Charts already initialized, updating with current data');
            this.updateKafkaDisplay();
        }
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
            topicCount: this.kafkaData ? new Set(this.kafkaData.map(m => m.topic)).size : 0,
            bufferSize: this.dataBuffer.size,
            totalBufferedPoints: Array.from(this.dataBuffer.values()).reduce((sum, arr) => sum + arr.length, 0),
            isInitialized: this.isInitialized,
            chartsInitialized: this.chartsInitialized
        };
    }

    // Get historical data for a specific topic
    getHistoricalData(topic, minutes = 5) {
        const data = this.dataBuffer.get(topic) || [];
        const cutoffTime = Date.now() - (minutes * 60 * 1000);
        return data.filter(point => point.timestamp >= cutoffTime);
    }

    // Clear old data manually (for debugging/testing)
    clearOldData() {
        this.cleanupOldData();
        console.log('Old data cleared manually');
    }
}

// Export for use in other modules
window.KafkaMetricsManager = KafkaMetricsManager;
// Kafka Pod Memory Usage Chart Module

class KafkaPodMemoryManager {
    constructor() {
        this.data = [];
        this.lastUpdate = null;
        this.updateInterval = 5 * 60 * 1000; // 5 minutes
        this.isUpdating = false;
        this.chartInstance = null;
        this.chartColors = {
            light: ['#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6', '#06b6d4'],
            dark: ['#60a5fa', '#f87171', '#4ade80', '#fbbf24', '#a78bfa', '#22d3ee']
        };

        // Test run filtering (reuse from KafkaMetricsManager)
        this.selectedTestRun = null;
        this.isTestFiltered = false;

        // Chart initialization state
        this.chartsInitialized = false;
    }

    // Initialize the chart
    async initialize() {
        console.log('Initializing Kafka pod memory chart...');

        // Wait for kafka manager to be ready before attaching listeners
        await this.waitForKafkaManager();

        this.attachEventListeners();
        await this.fetchData();
        this.startAutoUpdate();
    }

    // Wait for the Kafka manager to be initialized
    async waitForKafkaManager() {
        return new Promise((resolve) => {
            const checkKafkaManager = () => {
                if (window.realtimeManager && window.realtimeManager.kafkaManager && window.realtimeManager.kafkaManager.testRuns) {
                    console.log('KafkaPodMemoryManager: Kafka manager is ready');
                    resolve();
                } else {
                    console.log('KafkaPodMemoryManager: Waiting for kafka manager...');
                    setTimeout(checkKafkaManager, 100);
                }
            };
            checkKafkaManager();
        });
    }

    // Attach event listeners
    attachEventListeners() {
        // Refresh button if exists
        const refreshBtn = document.getElementById('kafka-pod-memory-refresh');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refresh();
            });
        }

        // Listen to the shared test run dropdown from Kafka metrics
        const testRunDropdown = document.getElementById('kafka-test-run-dropdown');
        if (testRunDropdown) {
            testRunDropdown.addEventListener('change', (event) => {
                this.onTestRunChange(event.target.value);
            });
        }

        // Listen to the clear filter button
        const clearBtn = document.getElementById('kafka-clear-test-filter');
        if (clearBtn) {
            clearBtn.addEventListener('click', () => {
                this.clearTestFilter();
            });
        }
    }

    // Fetch data from API
    async fetchData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showLoading();

            // Build API URL with optional time range and test_id parameters
            let apiUrl = '/api/clickhouse/kafka-pod-memory';

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
                this.data = result.data;
                this.lastUpdate = new Date();
                console.log('Kafka pod memory data updated:', this.data.length, 'data points, isTestFiltered:', this.isTestFiltered);
                this.updateChart();
                this.hideLoading();
            } else {
                console.error('Failed to fetch Kafka pod memory data:', result.message);
                this.showError('Failed to fetch data from server');
            }
        } catch (error) {
            console.error('Error fetching Kafka pod memory data:', error);
            this.showError('Network error while fetching data');
        } finally {
            this.isUpdating = false;
        }
    }

    // Handle test run selection
    onTestRunChange(testId) {
        console.log('KafkaPodMemoryManager: onTestRunChange called with testId:', testId);

        // Get test run data from the KafkaMetricsManager via realtimeManager
        if (window.realtimeManager && window.realtimeManager.kafkaManager) {
            const kafkaManager = window.realtimeManager.kafkaManager;

            // If test runs haven't loaded yet, wait a bit and try again
            if (!kafkaManager.testRuns || kafkaManager.testRuns.length === 0) {
                console.log('KafkaPodMemoryManager: Test runs not loaded yet, waiting...');
                setTimeout(() => this.onTestRunChange(testId), 500);
                return;
            }

            if (testId) {
                const selectedTest = kafkaManager.testRuns.find(test => test.test_id === testId || test.TestID === testId || test.testId === testId || test.TestId === testId);
                if (selectedTest) {
                    this.selectedTestRun = selectedTest;
                    this.isTestFiltered = true;
                    console.log('KafkaPodMemoryManager: Test run selected:', selectedTest.test_id || selectedTest.TestID);
                    this.fetchData();
                } else {
                    console.error('KafkaPodMemoryManager: Test run not found for ID:', testId);
                    console.log('Available test runs:', kafkaManager.testRuns.map(t => t.test_id || t.TestID || t.testId || t.TestId));
                }
            } else {
                this.clearTestFilter();
            }
        } else {
            console.error('KafkaPodMemoryManager: realtimeManager or kafkaManager not available');
        }
    }

    // Clear test filter
    clearTestFilter() {
        this.selectedTestRun = null;
        this.isTestFiltered = false;
        console.log('KafkaPodMemoryManager: Test filter cleared');
        this.fetchData();
        this.startAutoUpdate();
    }

    // Initialize ECharts instance
    initializeChart() {
        const chartContainer = document.getElementById('kafka-pod-memory-chart');
        if (!chartContainer) {
            console.warn('Chart container not found');
            return;
        }

        this.chartInstance = echarts.init(chartContainer);
        this.chartsInitialized = true;
        console.log('Kafka pod memory chart initialized');
    }

    // Update the chart with current data
    updateChart() {
        if (!this.chartInstance) {
            this.initializeChart();
        }

        if (!this.chartInstance || !this.data || this.data.length === 0) {
            this.showEmptyChart();
            return;
        }

        // Group data by pod
        const podData = {};
        this.data.forEach(point => {
            if (!podData[point.pod_name]) {
                podData[point.pod_name] = [];
            }
            podData[point.pod_name].push({
                timestamp: new Date(point.timestamp).getTime(),
                memoryUsage: point.memory_usage / (1024 * 1024 * 1024) // Convert to GB
            });
        });

        // Get theme colors
        const isDark = document.documentElement.classList.contains('dark');
        const colors = isDark ? this.chartColors.dark : this.chartColors.light;

        // Prepare series data
        const series = Object.keys(podData).map((podName, index) => {
            const data = podData[podName].sort((a, b) => a.timestamp - b.timestamp);

            return {
                name: podName,
                type: 'line',
                data: data.map(d => [d.timestamp, d.memoryUsage]),
                smooth: true,
                showSymbol: false,
                lineStyle: { width: 2 },
                itemStyle: { color: colors[index % colors.length] },
                areaStyle: { opacity: 0.1 }
            };
        });

        // Configure X-axis based on data mode
        let xAxisConfig = {
            type: 'time',
            name: 'Time (IST)',
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
            title: {
                text: this.isTestFiltered && this.selectedTestRun
                    ? `Kafka Pod Memory Usage (Test: ${this.selectedTestRun.test_name || 'Unnamed Test'})`
                    : 'Kafka Pod Memory Usage (Last 6 Hours)',
                left: 'center',
                textStyle: { fontSize: 16, fontWeight: 'normal' }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: { type: 'cross' },
                formatter: (params) => {
                    const date = new Date(params[0].value[0]);
                    let result = date.toLocaleString() + '<br/>';
                    params.forEach(param => {
                        result += `${param.seriesName}: ${param.value[1].toFixed(2)} GB<br/>`;
                    });
                    return result;
                }
            },
            legend: {
                data: Object.keys(podData),
                bottom: 0
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
                name: 'Memory Usage (GB)',
                nameLocation: 'middle',
                nameGap: 50
            },
            series: series
        };

        this.chartInstance.setOption(option, true);
        console.log('Kafka pod memory chart updated');
    }

    // Show empty state
    showEmptyChart() {
        if (!this.chartInstance) return;

        const option = {
            title: {
                text: 'No Data Available',
                left: 'center',
                top: 'center'
            },
            graphic: {
                type: 'text',
                left: 'center',
                top: 'middle',
                style: {
                    text: 'Waiting for Kafka pod memory data...',
                    fontSize: 12,
                    fill: '#64748b'
                }
            }
        };

        this.chartInstance.setOption(option, true);
    }

    // Show error state
    showError(message) {
        if (!this.chartInstance) return;

        const option = {
            title: {
                text: 'Error',
                left: 'center',
                top: 'center',
                textStyle: { color: '#ef4444' }
            },
            graphic: {
                type: 'text',
                left: 'center',
                top: 'middle',
                style: {
                    text: message,
                    fontSize: 12,
                    fill: '#ef4444'
                }
            }
        };

        this.chartInstance.setOption(option, true);
    }

    // Show loading state
    showLoading() {
        const loadingElement = document.getElementById('kafka-pod-memory-loading');
        if (loadingElement) {
            loadingElement.classList.remove('hidden');
        }
    }

    // Hide loading state
    hideLoading() {
        const loadingElement = document.getElementById('kafka-pod-memory-loading');
        if (loadingElement) {
            loadingElement.classList.add('hidden');
        }
    }

    // Start auto-update
    startAutoUpdate() {
        // Clear any existing interval
        if (this.updateIntervalId) {
            clearInterval(this.updateIntervalId);
        }

        // Only start auto-update if not in test filter mode
        if (!this.isTestFiltered) {
            this.updateIntervalId = setInterval(() => {
                this.fetchData();
            }, this.updateInterval);
        }
    }

    // Manual refresh
    async refresh() {
        await this.fetchData();
    }

    // Handle window resize
    handleResize() {
        if (this.chartInstance) {
            this.chartInstance.resize();
        }
    }

    // Initialize charts when section becomes visible
    onSectionVisible() {
        console.log('KafkaPodMemoryManager: onSectionVisible called, chartsInitialized:', this.chartsInitialized, 'isInitialized:', this.isInitialized);
        if (!this.chartsInitialized && this.isInitialized) {
            console.log('KafkaPodMemoryManager: Performance section became visible, initializing chart...');

            // Show loading state immediately
            setTimeout(() => {
                this.initializeChart();

                // Update display with current data
                if (this.data) {
                    console.log('KafkaPodMemoryManager: Updating display with existing data after chart initialization');
                    this.updateChart();
                } else {
                    // If no data yet, show loading state
                    console.log('KafkaPodMemoryManager: No data available, showing loading state');
                    this.showLoading();
                }
            }, 100); // Small delay to ensure DOM is ready
        } else if (this.chartsInitialized && this.data) {
            console.log('KafkaPodMemoryManager: Charts already initialized, updating with current data');
            this.updateChart();
        }
    }

    // Get current data
    getData() {
        return {
            data: this.data,
            lastUpdate: this.lastUpdate,
            podCount: this.data ? new Set(this.data.map(d => d.pod_name)).size : 0
        };
    }
}

// Export for use in other modules
window.KafkaPodMemoryManager = KafkaPodMemoryManager;
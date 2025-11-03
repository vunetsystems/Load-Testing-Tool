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
    }

    // Initialize the chart
    async initialize() {
        console.log('Initializing Kafka pod memory chart...');
        this.attachEventListeners();
        await this.fetchData();
        this.startAutoUpdate();
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
    }

    // Fetch data from API
    async fetchData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showLoading();

            const response = await fetch('/api/clickhouse/kafka-pod-memory');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.data = result.data;
                this.lastUpdate = new Date();
                console.log('Kafka pod memory data updated:', this.data.length, 'data points');
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

    // Initialize ECharts instance
    initializeChart() {
        const chartContainer = document.getElementById('kafka-pod-memory-chart');
        if (!chartContainer) {
            console.warn('Chart container not found');
            return;
        }

        this.chartInstance = echarts.init(chartContainer);
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

        const option = {
            title: {
                text: 'Kafka Pod Memory Usage (Last 6 Hours)',
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
            xAxis: {
                type: 'time',
                name: 'Time (IST)',
                nameLocation: 'middle',
                nameGap: 30,
                axisLabel: {
                    formatter: (value) => new Date(value).toLocaleTimeString()
                }
            },
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
        setInterval(() => {
            this.fetchData();
        }, this.updateInterval);
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
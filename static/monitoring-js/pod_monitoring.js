// Pod Monitoring JavaScript Module

class PodMonitoringManager {
    constructor() {
        this.podData = [];
        this.filteredData = [];
        this.podLogsCache = {}; // Cache logs by pod name
        this.yamlManager = new PodYAMLManager(); // YAML manager for pod YAML data
        this.lastUpdate = null;
        this.updateInterval = 30000; // 30 seconds
        this.isUpdating = false;
        this.currentPage = 1;
        this.itemsPerPage = 25;
        this.searchTerm = '';
        this.selectedNamespace = 'vsmaps';
        this.selectedTestID = ''; // Selected test ID for filtering
        this.testRuns = []; // Available test runs for dropdown
        // Sorting properties
        this.sortColumn = null;
        this.sortDirection = 'asc'; // 'asc' or 'desc'
        // Trend chart properties
        this.selectedPodForTrend = '';
        this.trendChart = null;
    }

    // Initialize pod monitoring
    async initialize() {
        console.log('Initializing pod monitoring...');
        this.attachEventListeners();
        await this.fetchTestRuns(); // Load test runs for dropdown
        await this.fetchPodLogs(); // Load logs first
        await this.fetchPodMonitoringData();
        this.initializeTrendChart();
        this.startAutoUpdate();
    }

    // Attach event listeners
    attachEventListeners() {
        // Search input
        const searchInput = document.getElementById('pod-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.searchTerm = e.target.value.toLowerCase();
                this.filterAndDisplayData();
            });
        }

        // Test ID filter
        const testIdFilter = document.getElementById('test-id-filter');
        if (testIdFilter) {
            testIdFilter.addEventListener('change', (e) => {
                this.selectedTestID = e.target.value;
                this.fetchPodMonitoringData();
                if (this.selectedPodForTrend) {
                    this.fetchPodTrendData();
                }
            });
        }

        // Namespace filter
        const namespaceFilter = document.getElementById('namespace-filter');
        if (namespaceFilter) {
            namespaceFilter.addEventListener('change', (e) => {
                this.selectedNamespace = e.target.value;
                this.fetchPodLogs(); // Reload logs for new namespace
                this.fetchPodMonitoringData();
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('pod-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refresh();
            });
        }

        // Sort button handlers
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('sort-btn') || e.target.closest('.sort-btn')) {
                const button = e.target.classList.contains('sort-btn') ? e.target : e.target.closest('.sort-btn');
                const column = button.dataset.column;
                this.handleSort(column);
            }
        });

        // Pod name click handler (opens modal)
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('pod-name-btn')) {
                const podName = e.target.dataset.podName;
                this.showPodDetailsModal(podName);
            }
        });

        // Close modal when clicking outside or close button
        const modal = document.getElementById('pod-details-modal');
        const closeBtn = document.getElementById('close-pod-modal');

        if (closeBtn) {
            closeBtn.addEventListener('click', () => {
                modal.classList.add('hidden');
                // Show sidebar and main content again
                const sidebar = document.querySelector('aside');
                const main = document.querySelector('main');
                if (sidebar) sidebar.classList.remove('hidden');
                if (main) main.classList.remove('hidden');
            });
        }

        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.classList.add('hidden');
                    // Show sidebar and main content again
                    const sidebar = document.querySelector('aside');
                    const main = document.querySelector('main');
                    if (sidebar) sidebar.classList.remove('hidden');
                    if (main) main.classList.remove('hidden');
                }
            });
        }

        // Pagination buttons
        const prevBtn = document.getElementById('pod-prev-btn');
        const nextBtn = document.getElementById('pod-next-btn');

        if (prevBtn) {
            prevBtn.addEventListener('click', () => {
                if (this.currentPage > 1) {
                    this.currentPage--;
                    this.displayCurrentPage();
                }
            });
        }

        if (nextBtn) {
            nextBtn.addEventListener('click', () => {
                const totalPages = Math.ceil(this.filteredData.length / this.itemsPerPage);
                if (this.currentPage < totalPages) {
                    this.currentPage++;
                    this.displayCurrentPage();
                }
            });
        }

        // Pod trend chart controls
        const podTrendSelect = document.getElementById('pod-trend-select');
        const podTrendRefresh = document.getElementById('pod-trend-refresh');

        if (podTrendSelect) {
            podTrendSelect.addEventListener('change', (e) => {
                this.selectedPodForTrend = e.target.value;
                if (this.selectedPodForTrend) {
                    this.fetchPodTrendData();
                } else {
                    this.showTrendNoData();
                }
            });
        }

        if (podTrendRefresh) {
            podTrendRefresh.addEventListener('click', () => {
                if (this.selectedPodForTrend) {
                    this.fetchPodTrendData();
                }
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
                console.log('Test runs loaded:', this.testRuns.length, 'runs');
                this.populateTestRunsDropdown();
            } else {
                console.error('Failed to fetch test runs:', result.message);
            }
        } catch (error) {
            console.error('Error fetching test runs:', error);
        }
    }

    // Populate test runs dropdown
    populateTestRunsDropdown() {
        const select = document.getElementById('test-id-filter');
        if (!select) return;

        // Clear existing options except the first one
        select.innerHTML = '<option value="">All Tests (Recent)</option>';

        // Add test runs as options
        this.testRuns.forEach(testRun => {
            const option = document.createElement('option');
            option.value = testRun.test_id;

            // Format display text
            const name = testRun.test_name || 'Unnamed Test';
            const startTime = new Date(testRun.start_time).toLocaleString();
            const status = testRun.status.charAt(0).toUpperCase() + testRun.status.slice(1);
            let displayText = `${name} - ${status} (${startTime})`;

            if (testRun.end_time) {
                const endTime = new Date(testRun.end_time).toLocaleString();
                displayText += ` to ${endTime}`;
            }

            option.textContent = displayText;
            select.appendChild(option);
        });

        console.log('Test runs dropdown populated with', this.testRuns.length, 'options');
    }


    // Fetch pod monitoring data from API
    async fetchPodMonitoringData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            let url = '/api/clickhouse/pod-monitoring';
            const params = new URLSearchParams();

            if (this.selectedNamespace !== 'All Namespaces') {
                params.append('namespace', this.selectedNamespace);
            }
            if (this.selectedTestID) {
                params.append('test_id', this.selectedTestID);
            }

            if (params.toString()) {
                url += '?' + params.toString();
            }

            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.podData = result.data;
                this.lastUpdate = new Date();
                console.log('Pod monitoring data updated:', this.podData.length, 'pods');
                this.updateLastUpdateTime();
                this.filterAndDisplayData();
                this.updateStatsCards();
                this.updatePodTrendDropdown();
            } else {
                console.error('Failed to fetch pod monitoring data:', result.message);
            }
        } catch (error) {
            console.error('Error fetching pod monitoring data:', error);
        } finally {
            this.isUpdating = false;
        }
    }

    // Fetch pod logs for the current namespace
    async fetchPodLogs() {
        try {
            const response = await fetch(`/api/clickhouse/pod-logs?namespace=${this.selectedNamespace}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                // Group logs by pod name for quick access
                this.podLogsCache = result.data.reduce((acc, log) => {
                    if (!acc[log.pod_name]) {
                        acc[log.pod_name] = [];
                    }
                    acc[log.pod_name].push(log);
                    return acc;
                }, {});
                console.log('Pod logs cached for', Object.keys(this.podLogsCache).length, 'pods');
            } else {
                console.error('Failed to fetch pod logs:', result.message);
                this.podLogsCache = {};
            }
        } catch (error) {
            console.error('Error fetching pod logs:', error);
            this.podLogsCache = {};
        }
    }

    // Update last update time display
    updateLastUpdateTime() {
        const lastUpdateElement = document.getElementById('pod-last-update');
        if (lastUpdateElement) {
            lastUpdateElement.textContent = this.lastUpdate ? this.lastUpdate.toLocaleTimeString() : 'Never';
        }
    }

    // Filter and display data
    filterAndDisplayData() {
        // Apply search filter
        this.filteredData = this.podData.filter(pod => {
            const matchesSearch = !this.searchTerm ||
                pod.pod_name.toLowerCase().includes(this.searchTerm) ||
                pod.namespace.toLowerCase().includes(this.searchTerm) ||
                pod.node_name.toLowerCase().includes(this.searchTerm);

            return matchesSearch;
        });

        // Apply sorting
        this.applySorting();

        // Reset to first page when filtering
        this.currentPage = 1;
        this.displayCurrentPage();
    }

    // Display current page of data
    displayCurrentPage() {
        const startIndex = (this.currentPage - 1) * this.itemsPerPage;
        const endIndex = startIndex + this.itemsPerPage;
        const pageData = this.filteredData.slice(startIndex, endIndex);

        this.renderPodTable(pageData);
        this.updatePagination();
    }

    // Render pod monitoring table
renderPodTable(pods) {
    const tbody = document.getElementById('pod-monitoring-body');
    if (!tbody) return;

    tbody.innerHTML = '';

    if (pods.length === 0) {
        const emptyRow = document.createElement('tr');
        emptyRow.innerHTML = `
            <td colspan="9" class="px-4 py-8 text-center text-slate-500">
                No pods found matching the current filters
            </td>
        `;
        tbody.appendChild(emptyRow);
        return;
    }

    pods.forEach(pod => {
        const row = document.createElement('tr');
        row.className =
            'border-b border-slate-200 last:border-b-0 hover:bg-slate-50 transition-colors';

        // Determine status color and text style
        let statusClass = 'status-unknown';
        if (pod.status) {
            const status = pod.status.toLowerCase();
            if (status.includes('running')) statusClass = 'status-running';
            else if (status.includes('pending')) statusClass = 'status-pending';
            else if (status.includes('failed') || status.includes('error'))
                statusClass = 'status-failed';
            else if (status.includes('succeeded')) statusClass = 'status-success';
        }

        // Format last seen
        const lastSeen = pod.last_seen ? new Date(pod.last_seen) : new Date();
        const timeAgo = this.getTimeAgo(lastSeen);

        // Handle nulls safely
        const cpuUsage = pod.cpu_usage != null && !isNaN(pod.cpu_usage) ? pod.cpu_usage : 0;
        const memoryUsage = pod.memory_usage != null && !isNaN(pod.memory_usage) ? pod.memory_usage : 0;
        const restarts = pod.restarts != null ? pod.restarts : 0;

        row.innerHTML = `
            <td class="h-[72px] px-4 py-2 text-sm">${pod.namespace || 'Unknown'}</td>
            <td class="h-[72px] px-4 py-2 text-sm font-medium">
                <button class="pod-name-btn text-indigo-600 hover:text-indigo-800 underline cursor-pointer" data-pod-name="${pod.pod_name || 'Unknown'}">
                    ${pod.pod_name || 'Unknown'}
                </button>
            </td>
            <td class="h-[72px] px-4 py-2 text-sm">${pod.node_name || 'Unknown'}</td>
            <td class="h-[72px] px-4 py-2 text-sm">
                <span class="status-badge ${statusClass}">
                    ${pod.status || 'Unknown'}
                </span>
            </td>
            <td class="h-[72px] px-4 py-2 text-sm">${pod.ready || 'N/A'}</td>
            <td class="h-[72px] px-4 py-2 text-sm">
                <div class="flex items-center gap-3">
                    <div class="w-20 overflow-hidden rounded-full bg-slate-100 border border-slate-300">
                        <div class="h-1.5 rounded-full bg-green-500" style="width: ${Math.min(cpuUsage, 100)}%;"></div>
                    </div>
                    <p class="font-medium">${cpuUsage.toFixed(1)}%</p>
                </div>
            </td>
            <td class="h-[72px] px-4 py-2 text-sm">
                <div class="flex items-center gap-3">
                    <div class="w-20 overflow-hidden rounded-full bg-slate-100 border border-slate-300">
                        <div class="h-1.5 rounded-full bg-blue-500" style="width: ${Math.min(memoryUsage, 100)}%;"></div>
                    </div>
                    <p class="font-medium">${memoryUsage.toFixed(1)}%</p>
                </div>
            </td>
            <td class="h-[72px] px-4 py-2 text-sm">${restarts}</td>
            <td class="h-[72px] px-4 py-2 text-sm text-slate-500">${timeAgo}</td>
        `;

        tbody.appendChild(row);
    });
}


    // Update pagination controls
    updatePagination() {
        const totalItems = this.filteredData.length;
        const totalPages = Math.ceil(totalItems / this.itemsPerPage);
        const startItem = (this.currentPage - 1) * this.itemsPerPage + 1;
        const endItem = Math.min(this.currentPage * this.itemsPerPage, totalItems);

        // Update pagination info
        const paginationInfo = document.getElementById('pod-pagination-info');
        if (paginationInfo) {
            paginationInfo.textContent = `Showing ${startItem} to ${endItem} of ${totalItems} results`;
        }

        // Update button states
        const prevBtn = document.getElementById('pod-prev-btn');
        const nextBtn = document.getElementById('pod-next-btn');

        if (prevBtn) {
            prevBtn.disabled = this.currentPage === 1;
        }
        if (nextBtn) {
            nextBtn.disabled = this.currentPage === totalPages;
        }
    }

    // Update statistics cards
    updateStatsCards() {
        const totalPods = this.podData.length;
        const runningPods = this.podData.filter(pod => pod.status && pod.status.toLowerCase().includes('running')).length;
        const validCpuPods = this.podData.filter(pod => pod.cpu_usage != null && !isNaN(pod.cpu_usage));
        const validMemoryPods = this.podData.filter(pod => pod.memory_usage != null && !isNaN(pod.memory_usage));
        const avgCpuUsage = validCpuPods.length > 0 ? validCpuPods.reduce((sum, pod) => sum + pod.cpu_usage, 0) / validCpuPods.length : 0;
        const avgMemoryUsage = validMemoryPods.length > 0 ? validMemoryPods.reduce((sum, pod) => sum + pod.memory_usage, 0) / validMemoryPods.length : 0;

        // Update cards
        this.updateCardValue('total-pods', totalPods);
        this.updateCardValue('running-pods', runningPods);
        this.updateCardValue('avg-cpu-usage', `${avgCpuUsage.toFixed(1)}%`);
        this.updateCardValue('avg-memory-usage', `${avgMemoryUsage.toFixed(1)}%`);

        // Update pod trend dropdown with current pods
        this.updatePodTrendDropdown();
    }

    // Update card value
    updateCardValue(cardId, value) {
        const element = document.getElementById(cardId);
        if (element) {
            element.textContent = value;
        }
    }

    // Get time ago string
    getTimeAgo(date) {
        const now = new Date();
        const diffInSeconds = Math.floor((now - date) / 1000);

        if (diffInSeconds < 60) return `${diffInSeconds}s ago`;
        if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}m ago`;
        if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}h ago`;
        return `${Math.floor(diffInSeconds / 86400)}d ago`;
    }

    // Start automatic updates
    startAutoUpdate() {
        setInterval(() => {
            this.fetchPodMonitoringData();
        }, this.updateInterval);
    }

    // Manual refresh
    async refresh() {
        await this.fetchPodMonitoringData();
    }

    // Initialize trend chart
    initializeTrendChart() {
        const chartContainer = document.getElementById('pod-trend-chart');
        if (!chartContainer) {
            console.warn('Pod trend chart container not found');
            return;
        }

        // Ensure container has dimensions
        chartContainer.style.width = '100%';
        chartContainer.style.height = '400px';

        this.trendChart = echarts.init(chartContainer);

        // Add window resize listener for responsiveness
        window.addEventListener('resize', () => {
            if (this.trendChart) {
                this.trendChart.resize();
            }
        });

        this.showTrendNoData();
    }

    // Update pod trend dropdown with current pods
    updatePodTrendDropdown() {
        const select = document.getElementById('pod-trend-select');
        if (!select) return;

        const currentValue = select.value;

        // Clear existing options except the first one
        select.innerHTML = '<option value="">Select a pod...</option>';

        // Add current pods as options
        this.podData.forEach(pod => {
            const option = document.createElement('option');
            option.value = pod.pod_name;
            option.textContent = `${pod.pod_name} (${pod.namespace})`;
            select.appendChild(option);
        });

        // Restore previous selection if it still exists
        if (currentValue && this.podData.some(pod => pod.pod_name === currentValue)) {
            select.value = currentValue;
        }

        console.log('Pod trend dropdown updated with', this.podData.length, 'pods');
    }

    // Fetch pod trend data
    async fetchPodTrendData() {
        if (!this.selectedPodForTrend) return;

        try {
            this.showTrendLoading();
            console.log('Fetching trend data for pod:', this.selectedPodForTrend, 'in namespace:', this.selectedNamespace, 'test_id:', this.selectedTestID);

            const params = new URLSearchParams({
                namespace: this.selectedNamespace,
                pod: this.selectedPodForTrend,
                hours: 24
            });
            if (this.selectedTestID) {
                params.append('test_id', this.selectedTestID);
            }
            const url = `/api/clickhouse/pod-trend?${params.toString()}`;

            const response = await fetch(url);
            console.log('Trend API response status:', response.status);

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            console.log('Trend API response:', result);

            if (result.success && result.data) {
                console.log('Rendering trend chart with', result.data.length, 'data points');
                this.renderTrendChart(result.data);
                this.hideTrendLoading();
            } else {
                console.error('Failed to fetch pod trend data:', result.message);
                this.showTrendNoData();
            }
        } catch (error) {
            console.error('Error fetching pod trend data:', error);
            this.showTrendNoData();
        }
    }

    // Render trend chart
    renderTrendChart(data) {
        if (!this.trendChart || !data.length) {
            console.log('No trend chart or no data, showing no data state');
            this.showTrendNoData();
            return;
        }

        console.log('Rendering trend chart with data:', data);

        const timestamps = data.map(item => new Date(item.timestamp).toLocaleString());
        const cpuData = data.map(item => item.cpu_usage || 0);
        const memoryData = data.map(item => item.memory_usage || 0);

        console.log('Timestamps:', timestamps.length, 'CPU data points:', cpuData.length, 'Memory data points:', memoryData.length);

        const option = {
            title: {
                text: `Resource Usage Trends - ${this.selectedPodForTrend}`,
                left: 'center',
                textStyle: {
                    fontSize: 16,
                    fontWeight: 'normal'
                }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: {
                    type: 'cross'
                },
                formatter: (params) => {
                    const timestamp = params[0].name;
                    let result = `<strong>${timestamp}</strong><br/>`;
                    params.forEach(param => {
                        result += `${param.seriesName}: ${param.value.toFixed(2)}%<br/>`;
                    });
                    return result;
                }
            },
            legend: {
                data: ['CPU Usage', 'Memory Usage'],
                top: 30
            },
            grid: {
                left: '3%',
                right: '4%',
                bottom: '15%',
                containLabel: true
            },
            xAxis: {
                type: 'category',
                boundaryGap: false,
                data: timestamps,
                axisLabel: {
                    rotate: 45,
                    fontSize: 10
                }
            },
            yAxis: {
                type: 'value',
                name: 'Usage (%)',
                nameLocation: 'middle',
                nameGap: 50
            },
            series: [
                {
                    name: 'CPU Usage',
                    type: 'line',
                    data: cpuData,
                    smooth: true,
                    lineStyle: {
                        color: '#3B82F6',
                        width: 2
                    },
                    itemStyle: {
                        color: '#3B82F6'
                    },
                    areaStyle: {
                        color: 'rgba(59, 130, 246, 0.1)'
                    }
                },
                {
                    name: 'Memory Usage',
                    type: 'line',
                    data: memoryData,
                    smooth: true,
                    lineStyle: {
                        color: '#10B981',
                        width: 2
                    },
                    itemStyle: {
                        color: '#10B981'
                    },
                    areaStyle: {
                        color: 'rgba(16, 185, 129, 0.1)'
                    }
                }
            ]
        };

        console.log('Setting chart option:', option);
        this.trendChart.setOption(option, true);
        console.log('Chart option set successfully');

        // Force resize after setting options to ensure proper dimensions
        setTimeout(() => {
            if (this.trendChart) {
                this.trendChart.resize();
            }
        }, 100);

        this.hideTrendLoading();
        this.showTrendChart();
    }

    // Show trend loading state
    showTrendLoading() {
        const loading = document.getElementById('pod-trend-loading');
        const noData = document.getElementById('pod-trend-no-data');
        const chart = document.getElementById('pod-trend-chart');

        if (loading) loading.classList.remove('hidden');
        if (noData) noData.classList.add('hidden');
        if (chart) chart.style.display = 'none';
    }

    // Hide trend loading state
    hideTrendLoading() {
        const loading = document.getElementById('pod-trend-loading');
        if (loading) loading.classList.add('hidden');
    }

    // Show trend no data state
    showTrendNoData() {
        const loading = document.getElementById('pod-trend-loading');
        const noData = document.getElementById('pod-trend-no-data');
        const chart = document.getElementById('pod-trend-chart');

        if (loading) loading.classList.add('hidden');
        if (noData) noData.classList.remove('hidden');
        if (chart) chart.style.display = 'none';
    }

    // Show trend chart
    showTrendChart() {
        const noData = document.getElementById('pod-trend-no-data');
        const chart = document.getElementById('pod-trend-chart');

        if (noData) noData.classList.add('hidden');
        if (chart) chart.style.display = 'block';
    }

    // Handle column sorting
    handleSort(column) {
        if (this.sortColumn === column) {
            // Toggle direction if same column
            this.sortDirection = this.sortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            // New column, default to ascending
            this.sortColumn = column;
            this.sortDirection = 'asc';
        }

        this.updateSortIndicators();
        this.filterAndDisplayData();
    }

    // Apply sorting to filtered data
    applySorting() {
        if (!this.sortColumn) return;

        this.filteredData.sort((a, b) => {
            let aValue = a[this.sortColumn];
            let bValue = b[this.sortColumn];

            // Handle null/undefined values
            if (aValue == null && bValue == null) return 0;
            if (aValue == null) return this.sortDirection === 'asc' ? 1 : -1;
            if (bValue == null) return this.sortDirection === 'asc' ? -1 : 1;

            let comparison = 0;

            // Type-specific sorting
            switch (this.sortColumn) {
                case 'cpu_usage':
                case 'memory_usage':
                case 'restarts':
                    // Numeric sorting
                    aValue = parseFloat(aValue) || 0;
                    bValue = parseFloat(bValue) || 0;
                    comparison = aValue - bValue;
                    break;

                case 'last_seen':
                    // Date sorting
                    const aDate = new Date(aValue);
                    const bDate = new Date(bValue);
                    comparison = aDate.getTime() - bDate.getTime();
                    break;

                default:
                    // String sorting (case-insensitive)
                    aValue = String(aValue).toLowerCase();
                    bValue = String(bValue).toLowerCase();
                    comparison = aValue.localeCompare(bValue);
                    break;
            }

            return this.sortDirection === 'asc' ? comparison : -comparison;
        });
    }

    // Update sort indicators in table headers
    updateSortIndicators() {
        // Reset all sort icons
        document.querySelectorAll('.sort-icon').forEach(icon => {
            icon.classList.remove('text-indigo-600');
            icon.classList.add('opacity-0');
        });

        // Update active sort column
        if (this.sortColumn) {
            const activeButton = document.querySelector(`[data-column="${this.sortColumn}"]`);
            if (activeButton) {
                const icon = activeButton.querySelector('.sort-icon');
                if (icon) {
                    icon.classList.remove('opacity-0');
                    icon.classList.add('text-indigo-600');
                    icon.textContent = this.sortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward';
                }
            }
        }
    }

    // Show pod details modal (full screen)
    showPodDetailsModal(podName) {
        const modal = document.getElementById('pod-details-modal');
        const podNameElement = document.getElementById('modal-pod-name');

        if (modal && podNameElement) {
            podNameElement.textContent = podName;
            modal.classList.remove('hidden');

            // Hide sidebar and main content for full-screen effect
            const sidebar = document.querySelector('aside');
            const main = document.querySelector('main');
            if (sidebar) sidebar.classList.add('hidden');
            if (main) main.classList.add('hidden');

            // Populate pod details
            this.populatePodDetails(podName);
        }
    }

    // Populate pod details in the full-screen modal
    populatePodDetails(podName) {
        const pod = this.podData.find(p => p.pod_name === podName);
        if (!pod) return;

        // Update basic info
        this.updateElement('pod-detail-name', pod.pod_name || 'Unknown');
        this.updateElement('pod-detail-namespace', pod.namespace || 'Unknown');
        this.updateElement('pod-detail-node', pod.node_name || 'Unknown');
        this.updateElement('pod-detail-status', pod.status || 'Unknown');
        this.updateElement('pod-detail-cpu', pod.cpu_usage != null ? `${pod.cpu_usage.toFixed(1)}%` : 'N/A');
        this.updateElement('pod-detail-memory', pod.memory_usage != null ? `${pod.memory_usage.toFixed(1)}%` : 'N/A');
        this.updateElement('pod-detail-restarts', pod.restarts != null ? pod.restarts : '0');
        this.updateElement('pod-detail-ready', pod.ready || 'N/A');

        // Update timestamps
        const lastSeen = pod.last_seen ? new Date(pod.last_seen) : new Date();
        this.updateElement('pod-detail-last-seen', this.getTimeAgo(lastSeen));
        this.updateElement('pod-detail-created', 'N/A'); // Would need additional API data
        this.updateElement('pod-detail-age', 'N/A'); // Would need additional API data

        // Update additional metrics
        this.updateElement('pod-detail-containers', 'N/A'); // Would need additional API data
        this.updateElement('pod-detail-phase', pod.status || 'Unknown');
        this.updateElement('pod-detail-qos', 'N/A'); // Would need additional API data

        // Initialize tabs
        this.initializePodDetailTabs();

        // Populate logs for this pod
        this.populatePodLogs(podName);

        // Populate events for this pod
        this.populatePodEvents(podName, pod.namespace);

        // Fetch YAML for this pod
        if (this.yamlManager) {
            this.yamlManager.fetchPodYAML(pod.namespace, podName).catch(error => {
                console.error('Error fetching pod YAML:', error);
            });
        }
    }

    // Populate pod logs in the logs tab
    populatePodLogs(podName) {
        const logsContent = document.getElementById('pod-logs-content');
        if (!logsContent) return;

        const podLogs = this.podLogsCache[podName] || [];

        if (podLogs.length === 0) {
            logsContent.innerHTML = '<div class="text-gray-400">No logs available for this pod</div>';
            return;
        }

        // Format and display logs
        const formattedLogs = podLogs.map(log => this.formatLogLine(log)).join('');
        logsContent.innerHTML = formattedLogs;

        // Auto-scroll to bottom
        logsContent.scrollTop = logsContent.scrollHeight;
    }

    // Fetch pod events for a specific pod
    async fetchPodEvents(podName, namespace) {
        try {
            const response = await fetch(`/api/clickhouse/pod-events?namespace=${namespace}&pod=${podName}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                // Return events for this specific pod (API now filters server-side)
                return result.data;
            } else {
                console.error('Failed to fetch pod events:', result.message);
                return [];
            }
        } catch (error) {
            console.error('Error fetching pod events:', error);
            return [];
        }
    }

    // Populate pod events in the events tab
    populatePodEvents(podName, namespace) {
        const eventsContent = document.getElementById('pod-events-content');
        if (!eventsContent) return;

        // Show loading state
        eventsContent.innerHTML = '<div class="text-center py-8 text-slate-500"><div class="inline-flex items-center gap-3"><div class="animate-spin rounded-full h-6 w-6 border-b-2 border-indigo-600"></div><span>Loading events...</span></div></div>';

        // Fetch and display events
        this.fetchPodEvents(podName, namespace).then(events => {
            if (events.length === 0) {
                eventsContent.innerHTML = '<div class="text-gray-400 text-center py-8">No events available for this pod</div>';
                return;
            }

            // Format and display events
            const formattedEvents = events.map(event => this.formatEventLine(event)).join('');
            eventsContent.innerHTML = formattedEvents;
        }).catch(error => {
            console.error('Error populating pod events:', error);
            eventsContent.innerHTML = '<div class="text-red-400 text-center py-8">Error loading events</div>';
        });
    }

    // Format a single event line
    formatEventLine(event) {
        const timestamp = new Date(event.timestamp).toLocaleString();
        const eventTypeColor = this.getEventTypeColor(event.event_type);
        const reasonColor = 'text-yellow-400';
        const messageColor = 'text-gray-300';

        return `
            <div class="event-line font-mono text-sm mb-2 p-3 bg-slate-800 rounded border-l-4 ${this.getEventBorderColor(event.event_type)}">
                <div class="flex items-center gap-2 mb-1">
                    <span class="text-gray-500">${timestamp}</span>
                    <span class="${eventTypeColor} font-bold px-2 py-1 rounded text-xs">${event.event_type}</span>
                    <span class="${reasonColor} font-semibold">${event.reason}</span>
                </div>
                <div class="${messageColor}">${event.message}</div>
                <div class="text-xs text-gray-500 mt-1">Host: ${event.host} | Namespace: ${event.namespace_name}</div>
            </div>
        `;
    }

    // Get color class for event type
    getEventTypeColor(eventType) {
        switch(eventType?.toLowerCase()) {
            case 'normal': return 'bg-green-600 text-white';
            case 'warning': return 'bg-yellow-600 text-white';
            default: return 'bg-gray-600 text-white';
        }
    }

    // Get border color for event type
    getEventBorderColor(eventType) {
        switch(eventType?.toLowerCase()) {
            case 'normal': return 'border-green-500';
            case 'warning': return 'border-yellow-500';
            default: return 'border-gray-500';
        }
    }

    // Format a single log line with colors
    formatLogLine(log) {
        const timestamp = new Date(log.timestamp).toLocaleString();
        const levelColor = this.getLogLevelColor(log.log_level);
        const podColor = 'text-blue-400';
        const containerColor = 'text-green-400';
        const messageColor = 'text-gray-300';

        return `
            <div class="log-line font-mono text-sm mb-1 leading-tight">
                <span class="text-gray-500 mr-2">${timestamp}</span>
                <span class="${levelColor} font-bold mr-2">[${log.log_level}]</span>
                <span class="${podColor} mr-2">${log.pod_name}</span>
                <span class="${containerColor} mr-2">(${log.container_name})</span>
                <span class="${messageColor}">${log.message}</span>
            </div>
        `;
    }

    // Get color class for log level
    getLogLevelColor(level) {
        switch(level?.toUpperCase()) {
            case 'ERROR': return 'text-red-400';
            case 'WARN': case 'WARNING': return 'text-yellow-400';
            case 'INFO': return 'text-blue-400';
            case 'DEBUG': return 'text-gray-400';
            default: return 'text-gray-300';
        }
    }

    // Update element text content
    updateElement(id, text) {
        const element = document.getElementById(id);
        if (element) {
            element.textContent = text;
        }
    }

    // Initialize pod detail tabs
    initializePodDetailTabs() {
        const tabButtons = document.querySelectorAll('.pod-detail-tab');
        const tabContents = document.querySelectorAll('.pod-detail-tab-content');

        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                // Remove active class from all tabs
                tabButtons.forEach(btn => {
                    btn.classList.remove('active');
                    btn.classList.remove('border-indigo-600', 'text-indigo-600');
                    btn.classList.add('text-slate-600', 'border-transparent');
                });

                // Hide all tab contents
                tabContents.forEach(content => content.classList.add('hidden'));

                // Activate clicked tab
                button.classList.add('active');
                button.classList.remove('text-slate-600', 'border-transparent');
                button.classList.add('text-indigo-600', 'border-indigo-600');

                // Show corresponding content
                const tabId = button.dataset.tab + '-tab';
                const content = document.getElementById(tabId);
                if (content) {
                    content.classList.remove('hidden');

                    // Special handling for YAML tab
                    if (button.dataset.tab === 'yaml') {
                        const podName = document.getElementById('modal-pod-name').textContent;
                        if (podName && this.yamlManager) {
                            this.yamlManager.displayYAML(podName);
                        }
                    }
                }
            });
        });
    }

    // Get current data
    getData() {
        const validCpuPods = this.podData.filter(pod => pod.cpu_usage != null && !isNaN(pod.cpu_usage));
        const validMemoryPods = this.podData.filter(pod => pod.memory_usage != null && !isNaN(pod.memory_usage));

        return {
            data: this.podData,
            filteredData: this.filteredData,
            lastUpdate: this.lastUpdate,
            stats: {
                totalPods: this.podData.length,
                runningPods: this.podData.filter(pod => pod.status && pod.status.toLowerCase().includes('running')).length,
                avgCpuUsage: validCpuPods.length > 0 ? validCpuPods.reduce((sum, pod) => sum + pod.cpu_usage, 0) / validCpuPods.length : 0,
                avgMemoryUsage: validMemoryPods.length > 0 ? validMemoryPods.reduce((sum, pod) => sum + pod.memory_usage, 0) / validMemoryPods.length : 0
            }
        };
    }
}

// Export for use in other modules
window.PodMonitoringManager = PodMonitoringManager;
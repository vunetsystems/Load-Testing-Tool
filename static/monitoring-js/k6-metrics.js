// K6 Monitoring JavaScript Module

class K6MonitoringManager {
    constructor() {
        this.k6Data = [];
        this.filteredData = [];
        this.k6DashboardData = [];
        this.k6LoginData = [];
        this.k6SuccessRate = null;
        this.k6MaxVus = null;
        this.lastUpdate = null;
        this.isUpdating = false;
        this.currentSearch = '';
        this.currentDashboardFilter = 'all';
        this.currentPage = 1;
        this.itemsPerPage = 10;
        this.dashboardCurrentPage = 1;
        this.dashboardItemsPerPage = 5;
        this.loginCurrentPage = 1;
        this.loginItemsPerPage = 5;
        this.currentChartView = 'all';
        this.chartInstance = null;
        // Sorting properties for panel table
        this.panelSortColumn = null;
        this.panelSortDirection = 'asc'; // 'asc' or 'desc'
        // Sorting properties for dashboard table
        this.dashboardSortColumn = null;
        this.dashboardSortDirection = 'asc';
        // Sorting properties for login table
        this.loginSortColumn = null;
        this.loginSortDirection = 'asc';
        // Test run filtering
        this.testRuns = [];
        this.selectedTestRun = null;
        this.isTestFiltered = false;
    }

    // Initialize K6 monitoring
    async initialize() {
        console.log('Initializing K6 monitoring...');
        this.attachEventListeners();

        // Load test runs for dropdown
        await this.fetchTestRuns();

        await this.fetchK6MaxVus();
        await this.fetchK6Data();
        await this.fetchK6DashboardData();
        await this.fetchK6SuccessRate();
        await this.fetchK6LoginData();
        this.initializeChart();
    }

    // Attach event listeners
    attachEventListeners() {
        // Search input
        const searchInput = document.getElementById('k6-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.currentSearch = e.target.value.toLowerCase();
                console.log('Search input changed:', this.currentSearch);
                this.filterAndDisplayData();
            });
        }

        // Dashboard filter
        const dashboardFilter = document.getElementById('k6-dashboard-filter');
        if (dashboardFilter) {
            dashboardFilter.addEventListener('change', (e) => {
                this.currentDashboardFilter = e.target.value;
                console.log('Dashboard filter changed:', this.currentDashboardFilter);
                this.filterAndDisplayData(); // Apply filter locally without refetching
                this.updateChart(); // Update chart when filter changes
            });
        }

        // Chart view buttons
        const chartViewBtns = document.querySelectorAll('.k6-chart-view-btn');
        chartViewBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                // Remove active class from all buttons
                chartViewBtns.forEach(b => {
                    b.classList.remove('bg-indigo-100', 'text-indigo-700', 'font-medium');
                    b.classList.add('bg-slate-100', 'text-slate-600', 'hover:bg-slate-200');
                });

                // Add active class to clicked button
                e.target.classList.remove('bg-slate-100', 'text-slate-600', 'hover:bg-slate-200');
                e.target.classList.add('bg-indigo-100', 'text-indigo-700', 'font-medium');

                this.currentChartView = e.target.dataset.view;
                this.updateChart();
            });
        });

        // Refresh button
        const refreshBtn = document.getElementById('k6-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refresh();
            });
        }

        // Test run dropdown event listeners
        const testRunDropdown = document.getElementById('k6-test-run-dropdown');
        const clearTestFilterBtn = document.getElementById('k6-clear-test-filter');

        if (testRunDropdown) {
            testRunDropdown.addEventListener('change', (event) => {
                this.onTestRunChange(event.target.value);
            });
        }

        if (clearTestFilterBtn) {
            clearTestFilterBtn.addEventListener('click', () => {
                this.clearTestFilter();
            });
        }

        // Pagination buttons
        const prevBtn = document.getElementById('k6-prev-btn');
        const nextBtn = document.getElementById('k6-next-btn');

        if (prevBtn) {
            prevBtn.addEventListener('click', () => {
                if (this.currentPage > 1) {
                    this.currentPage--;
                    this.displayCurrentPage();
                    this.updateChart(); // Update chart when page changes
                }   
            });
        }

        if (nextBtn) {
            nextBtn.addEventListener('click', () => {
                const totalPages = Math.ceil(this.filteredData.length / this.itemsPerPage);
                if (this.currentPage < totalPages) {
                    this.currentPage++;
                    this.displayCurrentPage();
                    this.updateChart(); // Update chart when page changes
                }
            });
        }

        // Dashboard pagination buttons
        const dashboardPrevBtn = document.getElementById('k6-dashboard-prev-btn');
        const dashboardNextBtn = document.getElementById('k6-dashboard-next-btn');

        if (dashboardPrevBtn) {
            dashboardPrevBtn.addEventListener('click', () => {
                if (this.dashboardCurrentPage > 1) {
                    this.dashboardCurrentPage--;
                    this.renderDashboardTable();
                }
            });
        }

        if (dashboardNextBtn) {
            dashboardNextBtn.addEventListener('click', () => {
                const totalPages = Math.ceil(this.k6DashboardData.length / this.dashboardItemsPerPage);
                if (this.dashboardCurrentPage < totalPages) {
                    this.dashboardCurrentPage++;
                    this.renderDashboardTable();
                }
            });
        }

        // Login pagination buttons
        const loginPrevBtn = document.getElementById('k6-login-prev-btn');
        const loginNextBtn = document.getElementById('k6-login-next-btn');

        if (loginPrevBtn) {
            loginPrevBtn.addEventListener('click', () => {
                if (this.loginCurrentPage > 1) {
                    this.loginCurrentPage--;
                    this.renderLoginTable();
                }
            });
        }

        if (loginNextBtn) {
            loginNextBtn.addEventListener('click', () => {
                const totalPages = Math.ceil(this.k6LoginData.length / this.loginItemsPerPage);
                if (this.loginCurrentPage < totalPages) {
                    this.loginCurrentPage++;
                    this.renderLoginTable();
                }
            });
        }

        // Sort button handlers for all tables
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('sort-btn') || e.target.closest('.sort-btn')) {
                const button = e.target.classList.contains('sort-btn') ? e.target : e.target.closest('.sort-btn');
                const column = button.dataset.column;
                const table = button.dataset.table; // Add data-table attribute to sort buttons

                if (table === 'panel') {
                    this.handlePanelSort(column);
                } else if (table === 'dashboard') {
                    this.handleDashboardSort(column);
                } else if (table === 'login') {
                    this.handleLoginSort(column);
                }
            }
        });
    }

    // Fetch K6 data from API
    async fetchK6Data() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showLoading();

            const params = new URLSearchParams();
            if (this.currentDashboardFilter && this.currentDashboardFilter !== 'all') {
                params.append('dashboard', this.currentDashboardFilter);
            }
            if (this.isTestFiltered && this.selectedTestRun) {
                params.append('test_id', this.selectedTestRun.test_id);
            }
            const url = `/api/clickhouse/k6-results${params.toString() ? '?' + params.toString() : ''}`;
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6Data = result.data;
                this.lastUpdate = new Date();
                console.log('K6 monitoring data updated:', this.k6Data.length, 'items, isTestFiltered:', this.isTestFiltered);
                console.log('Dashboard filter:', this.currentDashboardFilter);
                console.log('API URL:', url);
                this.updateLastUpdateTime();
                this.processData();
                this.filterAndDisplayData();
                this.hideLoading();
            } else {
                console.error('Failed to fetch K6 data:', result.message);
                this.showNoData();
            }
        } catch (error) {
            console.error('Error fetching K6 data:', error);
            this.showNoData();
        } finally {
            this.isUpdating = false;
        }
    }

    // Update last update time display
    updateLastUpdateTime() {
        const lastUpdateElement = document.getElementById('k6-last-update');
        if (lastUpdateElement) {
            lastUpdateElement.textContent = this.lastUpdate ? this.lastUpdate.toLocaleTimeString() : 'Never';
        }
    }

    // Fetch K6 max vus from API
    async fetchK6MaxVus() {
        try {
            const response = await fetch('/api/clickhouse/k6-max-vus');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6MaxVus = result.data.max_vus;
                console.log('K6 max vus updated:', this.k6MaxVus);
            } else {
                console.error('Failed to fetch K6 max vus:', result.message);
                this.k6MaxVus = null;
            }
        } catch (error) {
            console.error('Error fetching K6 max vus:', error);
            this.k6MaxVus = null;
        }
    }

    // Fetch K6 success rate from API
    async fetchK6SuccessRate() {
        try {
            const response = await fetch('/api/clickhouse/k6-success-rate');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6SuccessRate = result.data.avg_success_rate;
                console.log('K6 success rate updated:', this.k6SuccessRate);
                this.updateCardValue('k6-avg-success-rate', `${this.k6SuccessRate.toFixed(2)}%`);
            } else {
                console.error('Failed to fetch K6 success rate:', result.message);
                this.k6SuccessRate = null;
            }
        } catch (error) {
            console.error('Error fetching K6 success rate:', error);
            this.k6SuccessRate = null;
        }
    }

    // Process data for stats
    processData() {
        if (!this.k6Data.length) return;

        // Calculate stats
        const totalResults = this.k6Data.length;
        const avgP95 = this.k6Data.reduce((sum, item) => sum + item.p95_response_time, 0) / totalResults;
        const uniqueDashboards = [...new Set(this.k6Data.map(item => item.dashboard_name))];

        // Update stats display
        this.updateCardValue('k6-total-results', totalResults.toLocaleString());
        this.updateCardValue('k6-avg-p95', `${avgP95.toFixed(2)}ms`);
        this.updateCardValue('k6-vus', this.k6MaxVus ? this.k6MaxVus.toString() : 'N/A');
        this.updateCardValue('k6-dashboard', uniqueDashboards[0] || 'N/A');
    }

    // Filter and display data
    filterAndDisplayData() {
        // Apply filters
        this.filteredData = this.k6Data.filter(item => {
            const matchesSearch = !this.currentSearch ||
                (item.panel_name && item.panel_name.toLowerCase().includes(this.currentSearch)) ||
                (item.dashboard_name && item.dashboard_name.toLowerCase().includes(this.currentSearch));

            const matchesDashboard = this.currentDashboardFilter === 'all' ||
                this.currentDashboardFilter === '' ||
                (item.dashboard_name && item.dashboard_name === this.currentDashboardFilter);

            return matchesSearch && matchesDashboard;
        });

        // Apply sorting
        this.applyPanelSorting();

        // Reset to first page when filtering
        this.currentPage = 1;
        this.displayCurrentPage();
        console.log('Filtered data:', this.filteredData.length, 'items from', this.k6Data.length, 'total');
        console.log('Current search:', this.currentSearch);
        console.log('Current dashboard filter:', this.currentDashboardFilter);
    }

    // Initialize the ECharts instance
    initializeChart() {
        const chartContainer = document.getElementById('k6-panel-comparison-chart');
        if (!chartContainer) {
            console.warn('Chart container not found');
            return;
        }

        console.log('Initializing chart with container:', chartContainer);
        console.log('Container dimensions:', chartContainer.offsetWidth, 'x', chartContainer.offsetHeight);

        // Ensure container has dimensions
        chartContainer.style.width = '100%';
        chartContainer.style.height = '500px';
        chartContainer.style.minHeight = '400px';
        console.log('Set container dimensions to 100% x 500px');

        // Force layout recalculation
        chartContainer.offsetHeight;

        this.chartInstance = echarts.init(chartContainer);

        // Resize chart after initialization
        setTimeout(() => {
            this.chartInstance.resize();
            console.log('Chart resized');
        }, 100);
        console.log('ECharts instance created:', this.chartInstance);

        // Test with a simple chart first
        const testOption = {
            title: {
                text: 'Test Chart'
            },
            xAxis: {
                type: 'category',
                data: ['A', 'B', 'C']
            },
            yAxis: {
                type: 'value'
            },
            series: [{
                data: [1, 2, 3],
                type: 'bar'
            }]
        };

        this.chartInstance.setOption(testOption);
        console.log('Test chart rendered');

        // Wait a bit then update with real data
        setTimeout(() => {
            this.chartInstance.resize();
            this.updateChart();
        }, 1000);
    }

    // Update the chart with current data
    updateChart() {
        if (!this.chartInstance) {
            console.warn('Chart instance not initialized');
            return;
        }

        // Resize first
        this.chartInstance.resize();

        const chartData = this.prepareChartData();
        console.log('Chart data prepared:', chartData.length, 'items');
        const option = this.createChartOption(chartData);
        console.log('Chart option created:', option);
        this.chartInstance.setOption(option, true);
        console.log('Chart updated successfully');
    }

    // Get current page data for chart
    getCurrentPageData() {
        if (!this.filteredData.length) return [];

        const startIndex = (this.currentPage - 1) * this.itemsPerPage;
        const endIndex = Math.min(startIndex + this.itemsPerPage, this.filteredData.length);
        return this.filteredData.slice(startIndex, endIndex);
    }

    // Prepare data for the chart (only current page)
    prepareChartData() {
        const data = this.getCurrentPageData();
        console.log('Using current page data for chart:', data.length, 'items');

        if (!data.length) {
            console.log('No data for current page');
            return [];
        }

        // Group data by dashboard and calculate averages for current page only
        const dashboardStats = {};

        data.forEach(item => {
            const dashboard = item.dashboard_name || 'Unknown';
            if (!dashboardStats[dashboard]) {
                dashboardStats[dashboard] = {
                    totalResponseTime: 0,
                    panelCount: 0,
                    panels: []
                };
            }

            dashboardStats[dashboard].totalResponseTime += item.p95_response_time;
            dashboardStats[dashboard].panelCount += 1;
            dashboardStats[dashboard].panels.push({
                name: item.panel_name,
                responseTime: item.p95_response_time,
                status: item.panel_status
            });
        });

        console.log('Current page dashboard stats:', dashboardStats);

        // Calculate dashboard averages and panel deviations for current page
        const result = [];
        Object.keys(dashboardStats).forEach(dashboard => {
            const stats = dashboardStats[dashboard];
            const dashboardAvg = stats.totalResponseTime / stats.panelCount;

            stats.panels.forEach(panel => {
                const deviation = ((panel.responseTime - dashboardAvg) / dashboardAvg) * 100;
                const performance = this.categorizePerformance(deviation);

                result.push({
                    dashboard: dashboard,
                    panel: panel.name,
                    responseTime: panel.responseTime,
                    dashboardAvg: dashboardAvg,
                    deviation: deviation,
                    performance: performance,
                    status: panel.status
                });
            });
        });

        console.log('Processed current page chart data:', result.length, 'items');

        // Sort and filter based on current view
        if (this.currentChartView === 'outliers') {
            return result.filter(item => item.performance === 'worse');
        }

        // Return all current page data (up to 10 items)
        return result.sort((a, b) => b.responseTime - a.responseTime);
    }

    // Categorize performance based on deviation from dashboard average
    categorizePerformance(deviation) {
        if (deviation < -20) return 'better'; // More than 20% better
        if (deviation > 20) return 'worse';   // More than 20% worse
        return 'similar';                     // Within 20% of average
    }

    // Create ECharts option
    createChartOption(data) {
        console.log('Creating chart option with data:', data);

        if (!data || data.length === 0) {
            console.warn('No data provided to createChartOption');
            return {
                title: {
                    text: 'No Data Available',
                    left: 'center'
                }
            };
        }

        const categories = [...new Set(data.map(item => item.panel))];
        console.log('Chart categories:', categories);

        const dashboardAvgs = {};
        data.forEach(item => {
            dashboardAvgs[item.panel] = item.dashboardAvg;
        });

        const barData = data.map(item => ({
            value: item.responseTime,
            itemStyle: {
                color: this.getPerformanceColor(item.performance)
            }
        }));

        const lineData = categories.map(panel => dashboardAvgs[panel]);

        console.log('Bar data:', barData);
        console.log('Line data:', lineData);

        const option = {
            title: {
                text: `Current Page Panel Performance (${data.length} panels)`,
                left: 'center',
                textStyle: {
                    fontSize: 14,
                    fontWeight: 'normal'
                }
            },
            tooltip: {
                trigger: 'axis',
                axisPointer: {
                    type: 'shadow'
                },
                formatter: (params) => {
                    const item = data[params[0].dataIndex];
                    return `
                        <strong>${item.panel}</strong><br/>
                        Dashboard: ${item.dashboard}<br/>
                        Response Time: ${item.responseTime.toFixed(2)}ms<br/>
                        Dashboard Avg: ${item.dashboardAvg.toFixed(2)}ms<br/>
                        Deviation: ${item.deviation > 0 ? '+' : ''}${item.deviation.toFixed(1)}%<br/>
                        Status: ${item.status}
                    `;
                }
            },
            legend: {
                data: ['Panel Response Time', 'Dashboard Average'],
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
                data: categories,
                axisLabel: {
                    rotate: 45,
                    interval: 0,
                    fontSize: 10
                }
            },
            yAxis: {
                type: 'value',
                name: 'Response Time (ms)',
                nameLocation: 'middle',
                nameGap: 50
            },
            series: [
                {
                    name: 'Panel Response Time',
                    type: 'bar',
                    data: barData,
                    barWidth: '60%'
                },
                {
                    name: 'Dashboard Average',
                    type: 'line',
                    data: lineData,
                    lineStyle: {
                        color: '#6B7280',
                        width: 2,
                        type: 'dashed'
                    },
                    symbol: 'none'
                }
            ]
        };

        console.log('Final chart option:', option);
        return option;
    }

    // Get color based on performance category
    getPerformanceColor(performance) {
        switch (performance) {
            case 'better': return '#10B981'; // green
            case 'similar': return '#F59E0B'; // yellow
            case 'worse': return '#EF4444';  // red
            default: return '#6B7280';        // gray
        }
    }

    // Render the data table (shows current page only)
    renderTable() {
        const tbody = document.getElementById('k6-results-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!this.filteredData.length) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="8" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No results match the current filters
                </td>
            `;
            tbody.appendChild(emptyRow);
            this.updatePaginationControls();
            return;
        }

        // Calculate pagination
        const startIndex = (this.currentPage - 1) * this.itemsPerPage;
        const endIndex = Math.min(startIndex + this.itemsPerPage, this.filteredData.length);
        const currentPageData = this.filteredData.slice(startIndex, endIndex);

        currentPageData.forEach(item => {
            const row = document.createElement('tr');
            row.className = 'border-b border-border-light dark:border-border-dark hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

            // Format timestamp
            const timestamp = this.formatTimestamp(item.timestamp);

            // Get response time class
            const responseTimeClass = this.getResponseTimeClass(item.p95_response_time);

            row.innerHTML = `
                <td class="px-4 py-3 text-sm">${timestamp}</td>
                <td class="px-4 py-3 text-sm font-medium">${item.no_of_users}</td>
                <td class="px-4 py-3 text-sm">${item.test_id || 'N/A'}</td>
                <td class="px-4 py-3 text-sm">${item.time_filter}</td>
                <td class="px-4 py-3 text-sm">${item.panel_name}</td>
                <td class="px-4 py-3 text-sm">${item.dashboard_name}</td>
                <td class="px-4 py-3 text-sm">${item.panel_status}</td>
                <td class="px-4 py-3 text-sm">
                    <span class="font-medium ${responseTimeClass}">
                        ${item.p95_response_time.toFixed(2)}ms
                    </span>
                </td>
            `;

            tbody.appendChild(row);
        });

        this.updatePaginationControls();
    }

    // Display current page data
    displayCurrentPage() {
        this.renderTable();
    }

    // Update pagination controls
    updatePaginationControls() {
        const prevBtn = document.getElementById('k6-prev-btn');
        const nextBtn = document.getElementById('k6-next-btn');
        const paginationInfo = document.getElementById('k6-pagination-info');

        const totalItems = this.filteredData.length;
        const totalPages = Math.ceil(totalItems / this.itemsPerPage);
        const startItem = (this.currentPage - 1) * this.itemsPerPage + 1;
        const endItem = Math.min(this.currentPage * this.itemsPerPage, totalItems);

        // Update pagination info
        if (paginationInfo) {
            if (totalItems === 0) {
                paginationInfo.textContent = 'Showing 0 to 0 of 0 results';
            } else {
                paginationInfo.textContent = `Showing ${startItem} to ${endItem} of ${totalItems} results`;
            }
        }

        // Update button states
        if (prevBtn) {
            prevBtn.disabled = this.currentPage <= 1;
            prevBtn.classList.toggle('opacity-50', this.currentPage <= 1);
            prevBtn.classList.toggle('cursor-not-allowed', this.currentPage <= 1);
        }

        if (nextBtn) {
            nextBtn.disabled = this.currentPage >= totalPages;
            nextBtn.classList.toggle('opacity-50', this.currentPage >= totalPages);
            nextBtn.classList.toggle('cursor-not-allowed', this.currentPage >= totalPages);
        }
    }

    // Format timestamp for display
    formatTimestamp(timestamp) {
        try {
            // Return raw timestamp from ClickHouse without formatting
            return timestamp;
        } catch (e) {
            return timestamp;
        }
    }

    // Get CSS class for response time based on value
    getResponseTimeClass(responseTime) {
        if (responseTime < 100) return 'text-success dark:text-success-dark';
        if (responseTime < 500) return 'text-warning dark:text-warning-dark';
        return 'text-danger dark:text-danger-dark';
    }

    // Update card value
    updateCardValue(cardId, value) {
        const element = document.getElementById(cardId);
        if (element) {
            element.textContent = value;
        }
    }

    // Show loading state
    showLoading() {
        const loadingState = document.getElementById('k6-loading-state');
        const noDataState = document.getElementById('k6-no-data-state');

        if (loadingState) {
            loadingState.classList.remove('hidden');
        }
        if (noDataState) {
            noDataState.classList.add('hidden');
        }
    }

    // Hide loading state
    hideLoading() {
        const loadingState = document.getElementById('k6-loading-state');
        if (loadingState) {
            loadingState.classList.add('hidden');
        }
    }

    // Show no data state
    showNoData() {
        this.hideLoading();
        const noDataState = document.getElementById('k6-no-data-state');
        if (noDataState) {
            noDataState.classList.remove('hidden');
        }
    }

    // Fetch K6 dashboard data from API
    async fetchK6DashboardData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showDashboardLoading();

            let apiUrl = '/api/clickhouse/k6-dashboard-results';
            if (this.isTestFiltered && this.selectedTestRun) {
                apiUrl += `?test_id=${encodeURIComponent(this.selectedTestRun.test_id)}`;
            }

            const response = await fetch(apiUrl);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6DashboardData = result.data;
                console.log('K6 dashboard data updated:', this.k6DashboardData, 'isTestFiltered:', this.isTestFiltered);
                this.renderDashboardTable();
                this.hideDashboardLoading();
            } else {
                console.error('Failed to fetch K6 dashboard data:', result.message);
                this.showDashboardNoData();
            }
        } catch (error) {
            console.error('Error fetching K6 dashboard data:', error);
            this.showDashboardNoData();
        } finally {
            this.isUpdating = false;
        }
    }

    // Render the dashboard data table
    renderDashboardTable() {
        const tbody = document.getElementById('k6-dashboard-results-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!this.k6DashboardData.length) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="7" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No dashboard results available
                </td>
            `;
            tbody.appendChild(emptyRow);
            this.updateDashboardPaginationControls();
            return;
        }

        // Apply sorting
        this.applyDashboardSorting();

        // Calculate pagination
        const startIndex = (this.dashboardCurrentPage - 1) * this.dashboardItemsPerPage;
        const endIndex = Math.min(startIndex + this.dashboardItemsPerPage, this.k6DashboardData.length);
        const currentPageData = this.k6DashboardData.slice(startIndex, endIndex);

        currentPageData.forEach(item => {
            const row = document.createElement('tr');
            row.className = 'border-b border-border-light dark:border-border-dark hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

            // Format timestamp
            const timestamp = this.formatTimestamp(item.timestamp);

            // Get response time class
            const responseTimeClass = this.getResponseTimeClass(item.p95_response_time);

            row.innerHTML = `
                <td class="px-4 py-3 text-sm">${timestamp}</td>
                <td class="px-4 py-3 text-sm font-medium">${item.no_of_users}</td>
                <td class="px-4 py-3 text-sm">${item.test_id || 'N/A'}</td>
                <td class="px-4 py-3 text-sm">${item.time_filter}</td>
                <td class="px-4 py-3 text-sm">${item.dashboard_status}</td>
                <td class="px-4 py-3 text-sm">${item.dashboard_name}</td>
                <td class="px-4 py-3 text-sm">
                    <span class="font-medium ${responseTimeClass}">
                        ${item.p95_response_time.toFixed(2)}ms
                    </span>
                </td>
            `;

            tbody.appendChild(row);
        });

        this.updateDashboardPaginationControls();
    }

    // Show dashboard loading state
    showDashboardLoading() {
        const loadingState = document.getElementById('k6-dashboard-loading-state');
        const noDataState = document.getElementById('k6-dashboard-no-data-state');

        if (loadingState) {
            loadingState.classList.remove('hidden');
        }
        if (noDataState) {
            noDataState.classList.add('hidden');
        }
    }

    // Hide dashboard loading state
    hideDashboardLoading() {
        const loadingState = document.getElementById('k6-dashboard-loading-state');
        if (loadingState) {
            loadingState.classList.add('hidden');
        }
    }

    // Show dashboard no data state
    showDashboardNoData() {
        this.hideDashboardLoading();
        const noDataState = document.getElementById('k6-dashboard-no-data-state');
        if (noDataState) {
            noDataState.classList.remove('hidden');
        }
    }

    // Update dashboard pagination controls
    updateDashboardPaginationControls() {
        const prevBtn = document.getElementById('k6-dashboard-prev-btn');
        const nextBtn = document.getElementById('k6-dashboard-next-btn');
        const paginationInfo = document.getElementById('k6-dashboard-pagination-info');
        const totalItems = this.k6DashboardData.length;
        const totalPages = Math.ceil(totalItems / this.dashboardItemsPerPage);
        const startItem = (this.dashboardCurrentPage - 1) * this.dashboardItemsPerPage + 1;
        const endItem = Math.min(this.dashboardCurrentPage * this.dashboardItemsPerPage, totalItems);

        // Update pagination info
        if (paginationInfo) {
            if (totalItems === 0) {
                paginationInfo.textContent = 'Showing 0 to 0 of 0 results';
            } else {
                paginationInfo.textContent = `Showing ${startItem} to ${endItem} of ${totalItems} results`;
            }
        }

        // Update button states
        if (prevBtn) {
            prevBtn.disabled = this.dashboardCurrentPage <= 1;
            prevBtn.classList.toggle('opacity-50', this.dashboardCurrentPage <= 1);
            prevBtn.classList.toggle('cursor-not-allowed', this.dashboardCurrentPage <= 1);
        }

        if (nextBtn) {
            nextBtn.disabled = this.dashboardCurrentPage >= totalPages;
            nextBtn.classList.toggle('opacity-50', this.dashboardCurrentPage >= totalPages);
            nextBtn.classList.toggle('cursor-not-allowed', this.dashboardCurrentPage >= totalPages);
        }
    }

    // Update login pagination controls
    updateLoginPaginationControls() {
        const prevBtn = document.getElementById('k6-login-prev-btn');
        const nextBtn = document.getElementById('k6-login-next-btn');
        const paginationInfo = document.getElementById('k6-login-pagination-info');
        const totalItems = this.k6LoginData.length;
        const totalPages = Math.ceil(totalItems / this.loginItemsPerPage);
        const startItem = (this.loginCurrentPage - 1) * this.loginItemsPerPage + 1;
        const endItem = Math.min(this.loginCurrentPage * this.loginItemsPerPage, totalItems);

        // Update pagination info
        if (paginationInfo) {
            if (totalItems === 0) {
                paginationInfo.textContent = 'Showing 0 to 0 of 0 results';
            } else {
                paginationInfo.textContent = `Showing ${startItem} to ${endItem} of ${totalItems} results`;
            }
        }

        // Update button states
        if (prevBtn) {
            prevBtn.disabled = this.loginCurrentPage <= 1;
            prevBtn.classList.toggle('opacity-50', this.loginCurrentPage <= 1);
            prevBtn.classList.toggle('cursor-not-allowed', this.loginCurrentPage <= 1);
        }

        if (nextBtn) {
            nextBtn.disabled = this.loginCurrentPage >= totalPages;
            nextBtn.classList.toggle('opacity-50', this.loginCurrentPage >= totalPages);
            nextBtn.classList.toggle('cursor-not-allowed', this.loginCurrentPage >= totalPages);
        }
    }

    // Fetch K6 login data from API
    async fetchK6LoginData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showLoginLoading();

            let apiUrl = '/api/clickhouse/k6-login-results';
            if (this.isTestFiltered && this.selectedTestRun) {
                apiUrl += `?test_id=${encodeURIComponent(this.selectedTestRun.test_id)}`;
            }

            const response = await fetch(apiUrl);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6LoginData = result.data;
                console.log('K6 login data updated:', this.k6LoginData, 'isTestFiltered:', this.isTestFiltered);
                this.renderLoginTable();
                this.hideLoginLoading();
            } else {
                console.error('Failed to fetch K6 login data:', result.message);
                this.showLoginNoData();
            }
        } catch (error) {
            console.error('Error fetching K6 login data:', error);
            this.showLoginNoData();
        } finally {
            this.isUpdating = false;
        }
    }

    // Render the login data table
    renderLoginTable() {
        const tbody = document.getElementById('k6-login-results-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!this.k6LoginData.length) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="5" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No login test results available
                </td>
            `;
            tbody.appendChild(emptyRow);
            this.updateLoginPaginationControls();
            return;
        }

        // Apply sorting
        this.applyLoginSorting();

        // Calculate pagination
        const startIndex = (this.loginCurrentPage - 1) * this.loginItemsPerPage;
        const endIndex = Math.min(startIndex + this.loginItemsPerPage, this.k6LoginData.length);
        const currentPageData = this.k6LoginData.slice(startIndex, endIndex);

        currentPageData.forEach(item => {
            const row = document.createElement('tr');
            row.className = 'border-b border-border-light dark:border-border-dark hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50';

            // Format timestamp
            const timestamp = this.formatTimestamp(item.timestamp);

            // Get response time class
            const responseTimeClass = this.getResponseTimeClass(item.p95_response_time);

            row.innerHTML = `
                <td class="px-4 py-3 text-sm">${timestamp}</td>
                <td class="px-4 py-3 text-sm font-medium">${item.no_of_users}</td>
                <td class="px-4 py-3 text-sm">${item.test_id || 'N/A'}</td>
                <td class="px-4 py-3 text-sm">${item.test_name}</td>
                <td class="px-4 py-3 text-sm">
                    <span class="font-medium ${responseTimeClass}">
                        ${item.p95_response_time.toFixed(2)}ms
                    </span>
                </td>
            `;

            tbody.appendChild(row);
        });

        this.updateLoginPaginationControls();
    }

    // Show login loading state
    showLoginLoading() {
        const loadingState = document.getElementById('k6-login-loading-state');
        const noDataState = document.getElementById('k6-login-no-data-state');

        if (loadingState) {
            loadingState.classList.remove('hidden');
        }
        if (noDataState) {
            noDataState.classList.add('hidden');
        }
    }

    // Hide login loading state
    hideLoginLoading() {
        const loadingState = document.getElementById('k6-login-loading-state');
        if (loadingState) {
            loadingState.classList.add('hidden');
        }
    }

    // Show login no data state
    showLoginNoData() {
        this.hideLoginLoading();
        const noDataState = document.getElementById('k6-login-no-data-state');
        if (noDataState) {
            noDataState.classList.remove('hidden');
        }
    }

    // Manual refresh
    async refresh() {
        // Reset pagination to first page
        this.currentPage = 1;
        this.dashboardCurrentPage = 1;
        this.loginCurrentPage = 1;

        await this.fetchK6MaxVus();
        await this.fetchK6Data();
        await this.fetchK6DashboardData();
        await this.fetchK6SuccessRate();
        await this.fetchK6LoginData();

        // Update chart after refresh
        this.updateChart();
    }

    // Handle column sorting for panel table
    handlePanelSort(column) {
        if (this.panelSortColumn === column) {
            // Toggle direction if same column
            this.panelSortDirection = this.panelSortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            // New column, default to ascending
            this.panelSortColumn = column;
            this.panelSortDirection = 'asc';
        }

        this.updatePanelSortIndicators();
        this.filterAndDisplayData(); // This will trigger re-rendering with sorting
    }

    // Handle column sorting for dashboard table
    handleDashboardSort(column) {
        if (this.dashboardSortColumn === column) {
            this.dashboardSortDirection = this.dashboardSortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            this.dashboardSortColumn = column;
            this.dashboardSortDirection = 'asc';
        }

        this.updateDashboardSortIndicators();
        this.renderDashboardTable();
    }

    // Handle column sorting for login table
    handleLoginSort(column) {
        if (this.loginSortColumn === column) {
            this.loginSortDirection = this.loginSortDirection === 'asc' ? 'desc' : 'asc';
        } else {
            this.loginSortColumn = column;
            this.loginSortDirection = 'asc';
        }

        this.updateLoginSortIndicators();
        this.renderLoginTable();
    }

    // Apply sorting to panel filtered data
    applyPanelSorting() {
        if (!this.panelSortColumn) return;

        this.filteredData.sort((a, b) => {
            let aValue = a[this.panelSortColumn];
            let bValue = b[this.panelSortColumn];

            // Handle null/undefined values
            if (aValue == null && bValue == null) return 0;
            if (aValue == null) return this.panelSortDirection === 'asc' ? 1 : -1;
            if (bValue == null) return this.panelSortDirection === 'asc' ? -1 : 1;

            let comparison = 0;

            // Type-specific sorting
            switch (this.panelSortColumn) {
                case 'no_of_users':
                case 'p95_response_time':
                    // Numeric sorting
                    aValue = parseFloat(aValue) || 0;
                    bValue = parseFloat(bValue) || 0;
                    comparison = aValue - bValue;
                    break;

                case 'timestamp':
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

            return this.panelSortDirection === 'asc' ? comparison : -comparison;
        });
    }

    // Apply sorting to dashboard data
    applyDashboardSorting() {
        if (!this.dashboardSortColumn) return;

        this.k6DashboardData.sort((a, b) => {
            let aValue = a[this.dashboardSortColumn];
            let bValue = b[this.dashboardSortColumn];

            if (aValue == null && bValue == null) return 0;
            if (aValue == null) return this.dashboardSortDirection === 'asc' ? 1 : -1;
            if (bValue == null) return this.dashboardSortDirection === 'asc' ? -1 : 1;

            let comparison = 0;

            switch (this.dashboardSortColumn) {
                case 'no_of_users':
                case 'p95_response_time':
                    aValue = parseFloat(aValue) || 0;
                    bValue = parseFloat(bValue) || 0;
                    comparison = aValue - bValue;
                    break;

                case 'timestamp':
                    const aDate = new Date(aValue);
                    const bDate = new Date(bValue);
                    comparison = aDate.getTime() - bDate.getTime();
                    break;

                default:
                    aValue = String(aValue).toLowerCase();
                    bValue = String(bValue).toLowerCase();
                    comparison = aValue.localeCompare(bValue);
                    break;
            }

            return this.dashboardSortDirection === 'asc' ? comparison : -comparison;
        });
    }

    // Apply sorting to login data
    applyLoginSorting() {
        if (!this.loginSortColumn) return;

        this.k6LoginData.sort((a, b) => {
            let aValue = a[this.loginSortColumn];
            let bValue = b[this.loginSortColumn];

            if (aValue == null && bValue == null) return 0;
            if (aValue == null) return this.loginSortDirection === 'asc' ? 1 : -1;
            if (bValue == null) return this.loginSortDirection === 'asc' ? -1 : 1;

            let comparison = 0;

            switch (this.loginSortColumn) {
                case 'no_of_users':
                case 'p95_response_time':
                    aValue = parseFloat(aValue) || 0;
                    bValue = parseFloat(bValue) || 0;
                    comparison = aValue - bValue;
                    break;

                case 'timestamp':
                    const aDate = new Date(aValue);
                    const bDate = new Date(bValue);
                    comparison = aDate.getTime() - bDate.getTime();
                    break;

                default:
                    aValue = String(aValue).toLowerCase();
                    bValue = String(bValue).toLowerCase();
                    comparison = aValue.localeCompare(bValue);
                    break;
            }

            return this.loginSortDirection === 'asc' ? comparison : -comparison;
        });
    }

    // Update sort indicators for panel table
    updatePanelSortIndicators() {
        // Reset all sort icons in panel table
        document.querySelectorAll('.sort-btn[data-table="panel"] .sort-icon').forEach(icon => {
            icon.classList.remove('text-indigo-600');
            icon.classList.add('opacity-0');
        });

        // Update active sort column
        if (this.panelSortColumn) {
            const activeButton = document.querySelector(`[data-column="${this.panelSortColumn}"][data-table="panel"]`);
            if (activeButton) {
                const icon = activeButton.querySelector('.sort-icon');
                if (icon) {
                    icon.classList.remove('opacity-0');
                    icon.classList.add('text-indigo-600');
                    icon.textContent = this.panelSortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward';
                }
            }
        }
    }

    // Update sort indicators for dashboard table
    updateDashboardSortIndicators() {
        document.querySelectorAll('.sort-btn[data-table="dashboard"] .sort-icon').forEach(icon => {
            icon.classList.remove('text-indigo-600');
            icon.classList.add('opacity-0');
        });

        if (this.dashboardSortColumn) {
            const activeButton = document.querySelector(`[data-column="${this.dashboardSortColumn}"][data-table="dashboard"]`);
            if (activeButton) {
                const icon = activeButton.querySelector('.sort-icon');
                if (icon) {
                    icon.classList.remove('opacity-0');
                    icon.classList.add('text-indigo-600');
                    icon.textContent = this.dashboardSortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward';
                }
            }
        }
    }

    // Update sort indicators for login table
    updateLoginSortIndicators() {
        document.querySelectorAll('.sort-btn[data-table="login"] .sort-icon').forEach(icon => {
            icon.classList.remove('text-indigo-600');
            icon.classList.add('opacity-0');
        });

        if (this.loginSortColumn) {
            const activeButton = document.querySelector(`[data-column="${this.loginSortColumn}"][data-table="login"]`);
            if (activeButton) {
                const icon = activeButton.querySelector('.sort-icon');
                if (icon) {
                    icon.classList.remove('opacity-0');
                    icon.classList.add('text-indigo-600');
                    icon.textContent = this.loginSortDirection === 'asc' ? 'arrow_upward' : 'arrow_downward';
                }
            }
        }
    }

    // Fetch K6 runs for dropdown
    async fetchTestRuns() {
        try {
            const response = await fetch('/api/k6/dropdown');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.testRuns = result.data;
                this.populateTestRunDropdown();
                console.log('K6 runs loaded for K6:', this.testRuns.length);
            } else {
                console.error('Failed to fetch K6 runs:', result.message);
            }
        } catch (error) {
            console.error('Error fetching K6 runs:', error);
        }
    }

    // Populate dropdown with test runs
    populateTestRunDropdown() {
        const dropdown = document.getElementById('k6-test-run-dropdown');
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
        console.log('K6 onTestRunChange called with testId:', testId);
        if (!testId) {
            this.clearTestFilter();
            return;
        }

        const selectedTest = this.testRuns.find(test => test.test_id === testId);
        if (selectedTest) {
            this.selectedTestRun = selectedTest;
            this.isTestFiltered = true;

            // Update UI
            this.updateSelectedTestInfo();

            // Refresh data with test_id filter
            console.log('Fetching K6 data for test run:', selectedTest.test_id);
            this.refresh();

            console.log('Test run selected for K6:', selectedTest);
        } else {
            console.error('Test run not found for ID:', testId);
        }
    }

    // Clear test filter
    clearTestFilter() {
        this.selectedTestRun = null;
        this.isTestFiltered = false;

        // Reset dropdown
        const dropdown = document.getElementById('k6-test-run-dropdown');
        if (dropdown) {
            dropdown.value = '';
        }

        // Hide test info
        const infoDiv = document.getElementById('k6-selected-test-info');
        if (infoDiv) {
            infoDiv.classList.add('hidden');
        }

        // Refresh data without filter
        this.refresh();

        console.log('Test filter cleared for K6');
    }

    // Update selected test info display
    updateSelectedTestInfo() {
        const infoDiv = document.getElementById('k6-selected-test-info');
        const nameSpan = document.getElementById('k6-selected-test-name');
        const rangeSpan = document.getElementById('k6-selected-test-range');
        const durationSpan = document.getElementById('k6-selected-test-duration');

        if (!infoDiv || !this.selectedTestRun) return;

        nameSpan.textContent = this.selectedTestRun.test_name || 'Unnamed Test';
        rangeSpan.textContent = `${new Date(this.selectedTestRun.start_time).toLocaleString()} - ${new Date(this.selectedTestRun.end_time).toLocaleString()}`;
        durationSpan.textContent = this.selectedTestRun.duration;

        infoDiv.classList.remove('hidden');
    }

    // Get current data
    getData() {
        return {
            data: this.k6Data,
            filteredData: this.filteredData,
            lastUpdate: this.lastUpdate,
            stats: {
                totalResults: this.k6Data.length,
                avgP95: this.k6Data.length > 0 ? this.k6Data.reduce((sum, item) => sum + item.p95_response_time, 0) / this.k6Data.length : 0,
                uniqueVUs: [...new Set(this.k6Data.map(item => item.no_of_users))],
                uniqueDashboards: [...new Set(this.k6Data.map(item => item.dashboard_name))]
            }
        };
    }
}

// Export for use in other modules
window.K6MonitoringManager = K6MonitoringManager;
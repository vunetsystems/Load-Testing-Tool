// K6 Monitoring JavaScript Module

class K6MonitoringManager {
    constructor() {
        this.k6Data = [];
        this.filteredData = [];
        this.k6DashboardData = [];
        this.k6LoginData = [];
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
    }

    // Initialize K6 monitoring
    async initialize() {
        console.log('Initializing K6 monitoring...');
        this.attachEventListeners();
        await this.fetchK6MaxVus();
        await this.fetchK6Data();
        await this.fetchK6DashboardData();
        await this.fetchK6LoginData();
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
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('k6-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refresh();
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

        // Dashboard pagination buttons
        const dashboardNextBtn = document.getElementById('k6-dashboard-next-btn');

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
        const loginNextBtn = document.getElementById('k6-login-next-btn');

        if (loginNextBtn) {
            loginNextBtn.addEventListener('click', () => {
                const totalPages = Math.ceil(this.k6LoginData.length / this.loginItemsPerPage);
                if (this.loginCurrentPage < totalPages) {
                    this.loginCurrentPage++;
                    this.renderLoginTable();
                }
            });
        }
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
            const url = `/api/clickhouse/k6-results${params.toString() ? '?' + params.toString() : ''}`;
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6Data = result.data;
                this.lastUpdate = new Date();
                console.log('K6 monitoring data updated:', this.k6Data.length, 'items');
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

        // Reset to first page when filtering
        this.currentPage = 1;
        this.displayCurrentPage();
        console.log('Filtered data:', this.filteredData.length, 'items from', this.k6Data.length, 'total');
        console.log('Current search:', this.currentSearch);
        console.log('Current dashboard filter:', this.currentDashboardFilter);
    }

    // Render the data table (shows current page only)
    renderTable() {
        const tbody = document.getElementById('k6-results-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (!this.filteredData.length) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="6" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
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
                <td class="px-4 py-3 text-sm">${item.time_filter}</td>
                <td class="px-4 py-3 text-sm">${item.panel_name}</td>
                <td class="px-4 py-3 text-sm">${item.dashboard_name}</td>
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
            const date = new Date(timestamp);
            return date.toLocaleString();
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

            const response = await fetch('/api/clickhouse/k6-dashboard-results');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6DashboardData = result.data;
                console.log('K6 dashboard data updated:', this.k6DashboardData);
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
                <td colspan="5" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No dashboard results available
                </td>
            `;
            tbody.appendChild(emptyRow);
            this.updateDashboardPaginationControls();
            return;
        }

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
                <td class="px-4 py-3 text-sm">${item.time_filter}</td>
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
        if (nextBtn) {
            nextBtn.disabled = this.dashboardCurrentPage >= totalPages;
            nextBtn.classList.toggle('opacity-50', this.dashboardCurrentPage >= totalPages);
            nextBtn.classList.toggle('cursor-not-allowed', this.dashboardCurrentPage >= totalPages);
        }
    }

    // Update login pagination controls
    updateLoginPaginationControls() {
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

            const response = await fetch('/api/clickhouse/k6-login-results');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.k6LoginData = result.data;
                console.log('K6 login data updated:', this.k6LoginData);
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
                <td colspan="4" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No login test results available
                </td>
            `;
            tbody.appendChild(emptyRow);
            this.updateLoginPaginationControls();
            return;
        }

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
        await this.fetchK6LoginData();
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
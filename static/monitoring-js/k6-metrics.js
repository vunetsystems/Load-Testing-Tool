// K6 Monitoring JavaScript Module

class K6MonitoringManager {
    constructor() {
        this.k6Data = [];
        this.filteredData = [];
        this.lastUpdate = null;
        this.isUpdating = false;
        this.currentSearch = '';
        this.currentDashboardFilter = 'all';
    }

    // Initialize K6 monitoring
    async initialize() {
        console.log('Initializing K6 monitoring...');
        this.attachEventListeners();
        await this.fetchK6Data();
    }

    // Attach event listeners
    attachEventListeners() {
        // Search input
        const searchInput = document.getElementById('k6-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.currentSearch = e.target.value.toLowerCase();
                this.filterAndDisplayData();
            });
        }

        // Dashboard filter
        const dashboardFilter = document.getElementById('k6-dashboard-filter');
        if (dashboardFilter) {
            dashboardFilter.addEventListener('change', (e) => {
                this.currentDashboardFilter = e.target.value;
                this.fetchK6Data(); // Refetch data when dashboard changes
            });
        }

        // Refresh button
        const refreshBtn = document.getElementById('k6-refresh-btn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.refresh();
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
                console.log('K6 monitoring data updated:', this.k6Data);
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

    // Process data for stats
    processData() {
        if (!this.k6Data.length) return;

        // Calculate stats
        const totalResults = this.k6Data.length;
        const avgP95 = this.k6Data.reduce((sum, item) => sum + item.p95_response_time, 0) / totalResults;
        const uniqueVUs = [...new Set(this.k6Data.map(item => item.no_of_users))];
        const uniqueDashboards = [...new Set(this.k6Data.map(item => item.dashboard_name))];

        // Update stats display
        this.updateCardValue('k6-total-results', totalResults.toLocaleString());
        this.updateCardValue('k6-avg-p95', `${avgP95.toFixed(2)}ms`);
        this.updateCardValue('k6-vus', uniqueVUs.length > 1 ? `${uniqueVUs.join(', ')}` : uniqueVUs[0] || 'N/A');
        this.updateCardValue('k6-dashboard', uniqueDashboards[0] || 'N/A');
    }

    // Filter and display data
    filterAndDisplayData() {
        // Apply filters
        this.filteredData = this.k6Data.filter(item => {
            const matchesSearch = !this.currentSearch ||
                item.panel_name.toLowerCase().includes(this.currentSearch) ||
                item.dashboard_name.toLowerCase().includes(this.currentSearch);

            return matchesSearch;
        });

        this.renderTable();
    }

    // Render the data table
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
            return;
        }

        this.filteredData.forEach(item => {
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

    // Manual refresh
    async refresh() {
        await this.fetchK6Data();
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
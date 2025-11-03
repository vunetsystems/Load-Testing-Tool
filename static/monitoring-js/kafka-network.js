// Kafka Network Monitoring JavaScript Module

class KafkaNetworkManager {
    constructor() {
        this.data = [];
        this.currentPage = 1;
        this.itemsPerPage = 5;
        this.totalRecords = 0;
        this.totalPages = 0;
        this.lastUpdate = null;
        this.isUpdating = false;
    }

    // Initialize the network monitoring
    async initialize() {
        console.log('Initializing Kafka network monitoring...');
        this.attachEventListeners();
        await this.fetchData();
        this.startAutoUpdate();
    }

    // Attach event listeners
    attachEventListeners() {
        // Reload button
        const reloadBtn = document.getElementById('kafka-network-reload');
        if (reloadBtn) {
            reloadBtn.addEventListener('click', () => {
                this.reload();
            });
        }

        // Next button
        const nextBtn = document.getElementById('kafka-network-next');
        if (nextBtn) {
            nextBtn.addEventListener('click', () => {
                if (this.currentPage < this.totalPages) {
                    this.currentPage++;
                    this.fetchData();
                }
            });
        }

        // Previous button
        const prevBtn = document.getElementById('kafka-network-prev');
        if (prevBtn) {
            prevBtn.addEventListener('click', () => {
                if (this.currentPage > 1) {
                    this.currentPage--;
                    this.fetchData();
                }
            });
        }
    }

    // Fetch data from API
    async fetchData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            this.showLoading();

            const params = new URLSearchParams({
                page: this.currentPage,
                limit: this.itemsPerPage
            });

            const response = await fetch(`/api/clickhouse/kafka-network?${params}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.data = result.data.data || [];
                this.currentPage = result.data.page || 1;
                this.totalRecords = result.data.totalRecords || 0;
                this.totalPages = result.data.totalPages || 0;
                this.lastUpdate = new Date();

                console.log('Kafka network data updated:', this.data.length, 'records');
                this.updateDisplay();
                this.hideLoading();
            } else {
                console.error('Failed to fetch Kafka network data:', result.message);
                this.showError('Failed to fetch data from server');
            }
        } catch (error) {
            console.error('Error fetching Kafka network data:', error);
            this.showError('Network error while fetching data');
        } finally {
            this.isUpdating = false;
        }
    }

    // Update display
    updateDisplay() {
        this.updateTable();
        this.updatePagination();
        this.updateLastUpdateTime();
    }

    // Update table with data
    updateTable() {
        const tbody = document.getElementById('kafka-network-body');
        if (!tbody) return;

        tbody.innerHTML = '';

        if (this.data.length === 0) {
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = `
                <td colspan="8" class="px-4 py-8 text-center text-slate-500">
                    No network data available
                </td>
            `;
            tbody.appendChild(emptyRow);
            return;
        }

        this.data.forEach(item => {
            const row = document.createElement('tr');
            row.className = 'border-b border-slate-200 hover:bg-slate-50 transition-colors';

            const timestamp = new Date(item.timestamp).toLocaleString();

            row.innerHTML = `
                <td class="px-4 py-3 text-sm">${timestamp}</td>
                <td class="px-4 py-3 text-sm font-medium">${item.host_name}</td>
                <td class="px-4 py-3 text-sm">${item.interface_name}</td>
                <td class="px-4 py-3 text-sm">${item.bytes_sent_sec.toFixed(2)}</td>
                <td class="px-4 py-3 text-sm">${item.bytes_recv_sec.toFixed(2)}</td>
                <td class="px-4 py-3 text-sm">${item.packets_sent_sec.toFixed(2)}</td>
                <td class="px-4 py-3 text-sm">${item.packets_recv_sec.toFixed(2)}</td>
                <td class="px-4 py-3 text-sm">${item.drop_out_sec.toFixed(2)} / ${item.drop_in_sec.toFixed(2)}</td>
            `;

            tbody.appendChild(row);
        });
    }

    // Update pagination controls
    updatePagination() {
        const prevBtn = document.getElementById('kafka-network-prev');
        const nextBtn = document.getElementById('kafka-network-next');
        const pageInfo = document.getElementById('kafka-network-page-info');

        if (prevBtn) {
            prevBtn.disabled = this.currentPage <= 1;
            prevBtn.classList.toggle('opacity-50', this.currentPage <= 1);
        }

        if (nextBtn) {
            nextBtn.disabled = this.currentPage >= this.totalPages;
            nextBtn.classList.toggle('opacity-50', this.currentPage >= this.totalPages);
        }

        if (pageInfo) {
            pageInfo.textContent = `Page ${this.currentPage} of ${this.totalPages} (${this.totalRecords} total records)`;
        }
    }

    // Update last update time
    updateLastUpdateTime() {
        const lastUpdateElement = document.getElementById('kafka-network-last-update');
        if (lastUpdateElement) {
            lastUpdateElement.textContent = this.lastUpdate ? this.lastUpdate.toLocaleTimeString() : 'Never';
        }
    }

    // Show loading state
    showLoading() {
        const loadingElement = document.getElementById('kafka-network-loading');
        if (loadingElement) {
            loadingElement.classList.remove('hidden');
        }
    }

    // Hide loading state
    hideLoading() {
        const loadingElement = document.getElementById('kafka-network-loading');
        if (loadingElement) {
            loadingElement.classList.add('hidden');
        }
    }

    // Show error state
    showError(message) {
        this.hideLoading();
        const tbody = document.getElementById('kafka-network-body');
        if (tbody) {
            tbody.innerHTML = `
                <tr>
                    <td colspan="8" class="px-4 py-8 text-center text-red-500">
                        ${message}
                    </td>
                </tr>
            `;
        }
    }

    // Reload data
    async reload() {
        this.currentPage = 1; // Reset to first page on reload
        await this.fetchData();
    }

    // Start auto-update
    startAutoUpdate() {
        setInterval(() => {
            this.fetchData();
        }, 5 * 60 * 1000); // 5 minutes
    }

    // Get current data
    getData() {
        return {
            data: this.data,
            currentPage: this.currentPage,
            totalRecords: this.totalRecords,
            totalPages: this.totalPages,
            lastUpdate: this.lastUpdate
        };
    }
}

// Export for use in other modules
window.KafkaNetworkManager = KafkaNetworkManager;
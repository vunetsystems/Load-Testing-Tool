// Pod Monitoring JavaScript Module

class PodMonitoringManager {
    constructor() {
        this.podData = [];
        this.filteredData = [];
        this.lastUpdate = null;
        this.updateInterval = 30000; // 30 seconds
        this.isUpdating = false;
        this.currentPage = 1;
        this.itemsPerPage = 25;
        this.searchTerm = '';
        this.selectedNamespace = 'vsmaps';
    }

    // Initialize pod monitoring
    async initialize() {
        console.log('Initializing pod monitoring...');
        this.attachEventListeners();
        await this.fetchPodMonitoringData();
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

        // Namespace filter
        const namespaceFilter = document.getElementById('namespace-filter');
        if (namespaceFilter) {
            namespaceFilter.addEventListener('change', (e) => {
                this.selectedNamespace = e.target.value;
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
            });
        }

        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    modal.classList.add('hidden');
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
    }

    // Fetch pod monitoring data from API
    async fetchPodMonitoringData() {
        if (this.isUpdating) return;

        this.isUpdating = true;
        try {
            const namespaceParam = this.selectedNamespace !== 'All Namespaces' ? `?namespace=${this.selectedNamespace}` : '';
            const response = await fetch(`/api/clickhouse/pod-monitoring${namespaceParam}`);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const result = await response.json();
            if (result.success && result.data) {
                this.podData = result.data;
                this.lastUpdate = new Date();
                console.log('Pod monitoring data updated:', this.podData);
                this.updateLastUpdateTime();
                this.filterAndDisplayData();
                this.updateStatsCards();
            } else {
                console.error('Failed to fetch pod monitoring data:', result.message);
            }
        } catch (error) {
            console.error('Error fetching pod monitoring data:', error);
        } finally {
            this.isUpdating = false;
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
                <td colspan="9" class="px-4 py-8 text-center text-text-secondary-light dark:text-text-secondary-dark">
                    No pods found matching the current filters
                </td>
            `;
            tbody.appendChild(emptyRow);
            return;
        }

        pods.forEach(pod => {
            const row = document.createElement('tr');
            row.className = 'border-b border-border-light dark:border-border-dark last:border-b-0 hover:bg-background-light dark:hover:bg-background-dark transition-colors';

            // Determine status color
            let statusColor = 'bg-gray-500'; // Unknown
            if (pod.status) {
                const status = pod.status.toLowerCase();
                if (status.includes('running')) {
                    statusColor = 'bg-running';
                } else if (status.includes('pending')) {
                    statusColor = 'bg-pending';
                } else if (status.includes('failed') || status.includes('error')) {
                    statusColor = 'bg-failed';
                } else if (status.includes('succeeded')) {
                    statusColor = 'bg-success';
                }
            }

            // Format last seen time
            const lastSeen = pod.last_seen ? new Date(pod.last_seen) : new Date();
            const timeAgo = this.getTimeAgo(lastSeen);

            // Handle null values safely
            const cpuUsage = pod.cpu_usage != null && !isNaN(pod.cpu_usage) ? pod.cpu_usage : 0;
            const memoryUsage = pod.memory_usage != null && !isNaN(pod.memory_usage) ? pod.memory_usage : 0;
            const restarts = pod.restarts != null ? pod.restarts : 0;

            row.innerHTML = `
                <td class="h-[72px] px-4 py-2 text-sm">${pod.namespace || 'Unknown'}</td>
                <td class="h-[72px] px-4 py-2 text-sm font-medium">
                    <button class="pod-name-btn text-primary hover:text-primary-dark underline cursor-pointer" data-pod-name="${pod.pod_name || 'Unknown'}">
                        ${pod.pod_name || 'Unknown'}
                    </button>
                </td>
                <td class="h-[72px] px-4 py-2 text-sm">${pod.node_name || 'Unknown'}</td>
                <td class="h-[72px] px-4 py-2 text-sm">
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor}/20 text-gray-700 dark:text-gray-300">
                        ${pod.status || 'Unknown'}
                    </span>
                </td>
                <td class="h-[72px] px-4 py-2 text-sm">${pod.ready || 'N/A'}</td>
                <td class="h-[72px] px-4 py-2 text-sm">
                    <div class="flex items-center gap-3">
                        <div class="w-20 overflow-hidden rounded-full bg-border-light dark:bg-border-dark">
                            <div class="h-1.5 rounded-full bg-primary" style="width: ${Math.min(cpuUsage, 100)}%;"></div>
                        </div>
                        <p class="font-medium">${cpuUsage.toFixed(1)}%</p>
                    </div>
                </td>
                <td class="h-[72px] px-4 py-2 text-sm">
                    <div class="flex items-center gap-3">
                        <div class="w-20 overflow-hidden rounded-full bg-border-light dark:bg-border-dark">
                            <div class="h-1.5 rounded-full bg-primary" style="width: ${Math.min(memoryUsage, 100)}%;"></div>
                        </div>
                        <p class="font-medium">${memoryUsage.toFixed(1)}%</p>
                    </div>
                </td>
                <td class="h-[72px] px-4 py-2 text-sm">${restarts}</td>
                <td class="h-[72px] px-4 py-2 text-sm text-text-secondary-light dark:text-text-secondary-dark">${timeAgo}</td>
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

    // Show pod details modal
    showPodDetailsModal(podName) {
        const modal = document.getElementById('pod-details-modal');
        const podNameElement = document.getElementById('modal-pod-name');

        if (modal && podNameElement) {
            podNameElement.textContent = podName;
            modal.classList.remove('hidden');
        }
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
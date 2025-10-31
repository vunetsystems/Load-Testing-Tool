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
        // Sorting properties
        this.sortColumn = null;
        this.sortDirection = 'asc'; // 'asc' or 'desc'
    }

    // Initialize pod monitoring
    async initialize() {
        console.log('Initializing pod monitoring...');
        this.attachEventListeners();
        await this.fetchPodLogs(); // Load logs first
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
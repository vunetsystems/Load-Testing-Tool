// Main Application Manager - Refactored from monolithic script.js
class VuDataSimManager {
    constructor() {
        console.log('VuDataSimManager constructor called');
        this.isSimulationRunning = false;
        this.apiBaseUrl = ''; // Backend API base URL (empty for same origin)
        this.wsConnection = null;
        this.nodes = {}; // Store node data

        // Initialize modules
        this.dashboard = new DashboardManager(this);
        this.nodeManagement = new NodeManagement(this);
        this.binaryControl = new BinaryControl(this);
        this.clickHouseMetrics = new ClickHouseMetrics(this);
        this.o11ySources = new O11ySources(this);
        this.logsManager = new LogsManager(this);
        this.metricsManager = new MetricsManager(this);

        this.initializeComponents();
        this.bindEvents();
        this.dashboard.loadNodes(); // Load nodes from API
        this.startRealTimeUpdates();
        this.logsManager.populateNodeFilters(); // Populate filter dropdowns with real node names

        // Load o11y sources after DOM is ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                console.log('DOM loaded, about to load O11y sources...');
                this.o11ySources.loadO11ySources(); // Load available o11y sources
            });
        } else {
            console.log('DOM already loaded, loading O11y sources...');
            setTimeout(() => {
                this.o11ySources.loadO11ySources(); // Load available o11y sources
            }, 100);
        }
        console.log('VuDataSimManager initialization complete');
    }

    initializeComponents() {
        console.log('Initializing components...');
        // Cache DOM elements for better performance
        console.log('DOM elements for O11y sources:');
        console.log('- o11ySourcesContainer:', document.getElementById('o11y-sources-container'));
        console.log('- o11ySourcesDropdown:', document.getElementById('o11y-sources-dropdown'));
        console.log('- o11ySourcesOptions:', document.getElementById('o11y-sources-options'));
        console.log('- o11ySourcesList:', document.getElementById('o11y-sources-list'));

        this.elements = {
            logNodeFilter: document.getElementById('log-node'),
            logModuleFilter: document.getElementById('log-module'),
            logsContainer: document.getElementById('logs-container'),

                        // O11y source management elements
            o11ySourcesContainer: document.getElementById('o11y-sources-container'),
            o11ySourcesDropdown: document.getElementById('o11y-sources-dropdown'),
            o11ySourcesOptions: document.getElementById('o11y-sources-options'),
            o11ySourcesList: document.getElementById('o11y-sources-list'),
            o11ySourcesSearch: document.getElementById('o11y-sources-search'),
            o11ySourcesPlaceholder: document.getElementById('o11y-sources-placeholder'),
            o11ySourcesSelected: document.getElementById('o11y-sources-selected'),
            o11ySourcesArrow: document.getElementById('o11y-sources-arrow'),
            o11ySourcesSelectAll: document.getElementById('o11y-sources-select-all'),
            o11ySourcesClearAll: document.getElementById('o11y-sources-clear-all'),
            o11ySourcesCount: document.getElementById('o11y-sources-count'),
            epsSelect: document.getElementById('eps-select'),
            syncConfigsBtn: document.getElementById('sync-configs-btn'),
            syncStatusContainer: document.getElementById('sync-status-container'),
            syncSuccessMessage: document.getElementById('sync-success-message'),
            syncErrorMessage: document.getElementById('sync-error-message'),

            // Node management elements
            nodeManagementBtn: document.getElementById('node-management-btn'),
            nodeManagementModal: document.getElementById('node-management-modal'),
            modalBackdrop: document.getElementById('modal-backdrop'),
            closeNodeModal: document.getElementById('close-node-modal'),
            nodeName: document.getElementById('node-name'),
            nodeHost: document.getElementById('node-host'),
            nodeUser: document.getElementById('node-user'),
            nodeKeypath: document.getElementById('node-keypath'),
            nodeConfdir: document.getElementById('node-confdir'),
            nodeBindir: document.getElementById('node-bindir'),
            nodeDescription: document.getElementById('node-description'),
            nodeEnabled: document.getElementById('node-enabled'),
            addNodeBtn: document.getElementById('add-node-btn'),
            nodesTableBody: document.getElementById('nodes-table-body'),

            // Binary control elements
            binaryControlBtn: document.getElementById('binary-control-btn'),
            binaryControlModal: document.getElementById('binary-control-modal'),
            binaryModalBackdrop: document.getElementById('binary-modal-backdrop'),
            closeBinaryModal: document.getElementById('close-binary-modal'),
            binaryNodeSelect: document.getElementById('binary-node-select'),
            binaryPathInfo: document.getElementById('binary-path-info'),
            binaryPathDisplay: document.getElementById('binary-path-display'),
            binaryNameDisplay: document.getElementById('binary-name-display'),
            binaryStatusDisplay: document.getElementById('binary-status-display'),
            binaryCurrentStatus: document.getElementById('binary-current-status'),
            binaryCurrentPid: document.getElementById('binary-current-pid'),
            binaryCurrentStartTime: document.getElementById('binary-current-start-time'),
            binaryCurrentLastChecked: document.getElementById('binary-current-last-checked'),
            binaryCurrentProcessInfo: document.getElementById('binary-current-process-info'),
            refreshBinaryStatusBtn: document.getElementById('refresh-binary-status-btn'),
            startBinaryBtn: document.getElementById('start-binary-btn'),
            stopBinaryBtn: document.getElementById('stop-binary-btn'),
            binaryTimeoutInput: document.getElementById('binary-timeout-input'),
            binarySshOutput: document.getElementById('binary-ssh-output'),
            binaryAllStatusBody: document.getElementById('binary-all-status-body'),

            // Dashboard elements
            nodeStatusIndicatorsContainer: document.getElementById('node-status-indicators'),

            // Chart value elements
            cpuMemoryValue: document.getElementById('cpu-memory-value'),

            // ClickHouse metrics elements
            clickHouseMetricsBtn: document.getElementById('clickhouse-metrics-btn'),
            clickHouseMetricsModal: document.getElementById('clickhouse-metrics-modal'),
            clickHouseModalBackdrop: document.getElementById('clickhouse-modal-backdrop'),
            closeClickHouseModal: document.getElementById('close-clickhouse-modal'),
            refreshClickHouseMetricsBtn: document.getElementById('refresh-clickhouse-metrics-btn'),
            clickHouseStatus: document.getElementById('clickhouse-status'),
            clickHouseLastUpdate: document.getElementById('clickhouse-last-update'),
            kafkaMetricsTable: document.getElementById('kafka-metrics-table'),
            systemMetricsTable: document.getElementById('system-metrics-table'),
            databaseMetricsTable: document.getElementById('database-metrics-table'),
            containerMetricsTable: document.getElementById('container-metrics-table'),
            podResourceMetricsTable: document.getElementById('pod-resource-metrics-table'),
            podStatusMetricsTable: document.getElementById('pod-status-metrics-table'),
            topPodMemoryMetricsTable: document.getElementById('top-pod-memory-metrics-table'),
            kafkaTopicMetricsTable: document.getElementById('kafka-topic-metrics-table'),
            nodeFilterSelect: document.getElementById('node-filter-select'),

            // Real-time status
        };

        // Initialize node data (will be loaded from API with real data only)
        this.nodeData = {};

        // SSH status cache (updated hourly)
        this.sshStatuses = {};
        this.lastSshStatusUpdate = 0;

        // Legacy property for backward compatibility (browser cache)
        this.nodeManagementSection = null;

        // Real-time log entries (will be loaded from API)
        this.logEntries = [];
        this.lastLogUpdate = 0;
    }

    bindEvents() {
        // Button event listeners

        // O11y source management event listeners
        this.elements.syncConfigsBtn?.addEventListener('click', () => this.o11ySources.syncConfigs());

        // New custom multi-select event listeners
        this.elements.o11ySourcesDropdown?.addEventListener('click', () => this.o11ySources.toggleO11ySourcesDropdown());
        this.elements.o11ySourcesSearch?.addEventListener('input', (e) => this.o11ySources.filterO11ySources(e.target.value));
        this.elements.o11ySourcesSelectAll?.addEventListener('click', () => this.o11ySources.selectAllO11ySources());
        this.elements.o11ySourcesClearAll?.addEventListener('click', () => this.o11ySources.clearAllO11ySources());

        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            if (!this.elements.o11ySourcesContainer?.contains(e.target)) {
                this.o11ySources.closeO11ySourcesDropdown();
            }
        });

        // Node management event listeners
        console.log('Node Management Button element:', this.elements.nodeManagementBtn);
        console.log('Node Management Modal element:', this.elements.nodeManagementModal);

        if (this.elements.nodeManagementBtn) {
            this.elements.nodeManagementBtn.addEventListener('click', () => {
                console.log('Node Management button clicked!');
                this.nodeManagement.openNodeManagementModal();
            });
        } else {
            console.error('Node Management button not found!');
        }

        // Binary control event listeners
        console.log('Binary Control Button element:', this.elements.binaryControlBtn);
        console.log('Binary Control Modal element:', this.elements.binaryControlModal);

        if (this.elements.binaryControlBtn) {
            this.elements.binaryControlBtn.addEventListener('click', () => {
                console.log('Binary Control button clicked!');
                this.binaryControl.openBinaryControlModal();
            });
        } else {
            console.error('Binary Control button not found!');
        }

        // ClickHouse metrics event listeners
        console.log('ClickHouse Metrics Button element:', this.elements.clickHouseMetricsBtn);
        console.log('ClickHouse Metrics Modal element:', this.elements.clickHouseMetricsModal);

        if (this.elements.clickHouseMetricsBtn) {
            this.elements.clickHouseMetricsBtn.addEventListener('click', () => {
                console.log('ClickHouse Metrics button clicked!');
                this.clickHouseMetrics.openClickHouseMetricsModal();
            });
        } else {
            console.error('ClickHouse Metrics button not found!');
        }

        // Modal event listeners
        this.elements.closeNodeModal?.addEventListener('click', () => this.nodeManagement.closeNodeManagementModal());
        this.elements.modalBackdrop?.addEventListener('click', () => this.nodeManagement.closeNodeManagementModal());

        // Binary control modal event listeners
        this.elements.closeBinaryModal?.addEventListener('click', () => this.binaryControl.closeBinaryControlModal());
        this.elements.binaryModalBackdrop?.addEventListener('click', () => this.binaryControl.closeBinaryControlModal());

        // ClickHouse metrics modal event listeners
        this.elements.closeClickHouseModal?.addEventListener('click', () => this.clickHouseMetrics.closeClickHouseMetricsModal());
        this.elements.clickHouseModalBackdrop?.addEventListener('click', () => this.clickHouseMetrics.closeClickHouseMetricsModal());
        this.elements.refreshClickHouseMetricsBtn?.addEventListener('click', () => this.clickHouseMetrics.refreshClickHouseMetrics());
        this.elements.nodeFilterSelect?.addEventListener('change', () => this.clickHouseMetrics.filterTopPodMemoryMetrics());

        // Binary control action listeners
        this.elements.refreshBinaryStatusBtn?.addEventListener('click', () => this.binaryControl.refreshBinaryStatus());
        this.elements.startBinaryBtn?.addEventListener('click', () => this.binaryControl.startBinary());
        this.elements.stopBinaryBtn?.addEventListener('click', () => this.binaryControl.stopBinary());
        this.elements.binaryNodeSelect?.addEventListener('change', () => this.binaryControl.onBinaryNodeSelectChange());

        // Add get binary output button listener
        const getOutputBtn = document.getElementById('get-binary-output-btn');
        if (getOutputBtn) {
            getOutputBtn.addEventListener('click', () => this.binaryControl.getBinaryOutput());
        }

        // Close modal on Escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                if (!this.elements.nodeManagementModal.classList.contains('hidden')) {
                    this.nodeManagement.closeNodeManagementModal();
                }
                if (!this.elements.binaryControlModal.classList.contains('hidden')) {
                    this.binaryControl.closeBinaryControlModal();
                }
                if (!this.elements.clickHouseMetricsModal.classList.contains('hidden')) {
                    this.clickHouseMetrics.closeClickHouseMetricsModal();
                }
            }
        });

        this.elements.addNodeBtn?.addEventListener('click', () => this.nodeManagement.addNode());

        // Edit node form (initially hidden)
        this.isEditMode = false;
        this.editNodeName = null;


        // Log filtering
        this.elements.logNodeFilter?.addEventListener('change', () => this.logsManager.filterLogs());
        this.elements.logModuleFilter?.addEventListener('change', () => this.logsManager.filterLogs());

        // Real-time updates - Update every 3 seconds with real data
        setInterval(() => {
            this.updateMetrics();
            this.updateNodeStatusIndicators(); // Ensure status dots stay in sync
            // Note: updateDashboardDisplay() is now called hourly for SSH status
        }, 3000);

        // Cluster metrics updates - Update every 30 seconds
        setInterval(async () => {
            try {
                const metricsResponse = await this.callAPI('/api/cluster/metrics');
                console.log('=== CLUSTER METRICS API RESPONSE ===');
                console.log('Full response:', metricsResponse);
                console.log('Response data:', metricsResponse.data);
                console.log('Response success:', metricsResponse.success);
                if (metricsResponse.success && metricsResponse.data) {
                    this.dashboard.clusterMetrics = metricsResponse.data;
                    console.log('Assigned clusterMetrics:', this.dashboard.clusterMetrics);
                    this.dashboard.updateClusterTableOnly(); // Update only the metrics columns
                } else {
                    console.log('No valid metrics data in response');
                }
            } catch (error) {
                console.error('Error fetching cluster metrics:', error);
            }
        }, 30000); // 30 seconds


        // SSH status updates - Update every hour
        setInterval(() => {
            console.log('Hourly SSH status check triggered');
            this.dashboard.updateSSHStatuses();
        }, 60 * 60 * 1000); // 1 hour in milliseconds

        // Load logs initially and set up polling for live logs
        this.logsManager.loadLogs();
        setInterval(() => {
            this.logsManager.loadLogs();
        }, 5000); // Update logs every 5 seconds
    }

    updateMetrics() {
        // Only update display - no simulation
        // Real metrics are collected by the backend via SSH
        // Note: updateDashboardDisplay() is now called hourly for SSH status
    }

    startRealTimeUpdates() {
        // Initial display - fetch SSH status immediately on startup
        //this.updateDashboardDisplay();
        this.updateNodeStatusIndicators();
        this.nodeManagement.refreshNodesTable();
        this.logsManager.loadLogs(); // Load real logs instead of static ones
        this.logsManager.displayLogs(this.logEntries);

        // Set up WebSocket connection for real-time updates
        this.setupWebSocket();


        setInterval(() => {
            this.dashboard.fetchFinalVuDataSimMetrics();
        }, 3000);
    }

    setupWebSocket() {
        // In a real implementation, this would connect to the backend WebSocket
        // For now, we'll simulate real-time updates with intervals
        console.log('WebSocket connection would be established here');
    }

    async callAPI(endpoint, method = 'GET', data = null) {
        try {
            const config = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                },
            };

            if (data) {
                config.body = JSON.stringify(data);
            }

            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, config);

            // Always get the response body, even for error status codes
            const responseData = await response.json();

            // Check for HTTP error status codes and throw error for them
            if (!response.ok) {
                const errorMessage = responseData.message || `HTTP ${response.status}: ${response.statusText}`;
                const error = new Error(errorMessage);
                error.status = response.status;
                error.responseData = responseData;
                throw error;
            }

            // Return the actual response data for successful responses
            return responseData;
        } catch (error) {
            console.error('API call failed:', error);

            // Only return mock response for actual network errors, not HTTP error responses
            if (error.message.includes('fetch') || error.message.includes('network') || error.message.includes('CORS')) {
                console.warn('Network error detected, returning mock response:', error.message);
                return {
                    success: true,
                    message: 'Mock API response - Network error',
                    data: {}
                };
            }

            // For HTTP errors (including 500 errors), re-throw the error so calling code can handle it
            throw error;
        }
    }

    async refreshDashboard() {
        try {
            // Reload nodes from API
            await this.dashboard.loadNodes();
            // Also refresh the nodes table if modal is open
            if (this.elements.nodeManagementModal && !this.elements.nodeManagementModal.classList.contains('hidden')) {
                this.nodeManagement.refreshNodesTable();
            }
        } catch (error) {
            console.error('Error refreshing dashboard:', error);
        }
    }

    updateNodeStatusIndicators() {
        // Update node status indicators based on real node data
        const nodeIds = Object.keys(this.nodeData);
        const container = this.elements.nodeStatusIndicatorsContainer;
        container.innerHTML = '';

        nodeIds.forEach((nodeId, index) => {
            const node = this.nodeData[nodeId];
            const statusElement = document.createElement('span');
            statusElement.className = `h-4 w-4 rounded-full ${node.status === 'active' ? 'bg-success animate-node-pulse' : 'bg-danger'}`;
            statusElement.style.animationDelay = `${index * 0.2}s`;
            statusElement.title = `${nodeId}: ${node.status === 'active' ? 'Active' : 'Inactive'}`;

            container.appendChild(statusElement);
        });
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 px-6 py-3 rounded-lg shadow-lg z-50 animate-slide-down ${type === 'success' ? 'bg-success text-white' :
                type === 'error' ? 'bg-danger text-white' :
                    type === 'warning' ? 'bg-yellow-500 text-white' :
                        'bg-primary text-white'
            }`;
        notification.textContent = message;

        document.body.appendChild(notification);

        // Remove after 3 seconds
        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.vuDataSimManager = new VuDataSimManager();
});

// Test function to manually toggle node management
window.testNodeManagement = function () {
    console.log('Test function called');
    if (window.vuDataSimManager) {
        window.vuDataSimManager.nodeManagement.openNodeManagementModal();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Test function to manually load O11y sources
window.testLoadO11ySources = function() {
    console.log('Manual test: Loading O11y sources...');
    if (window.vuDataSimManager) {
        window.vuDataSimManager.o11ySources.loadO11ySources();
    } else {
        console.error('vuDataSimManager not initialized');
    }
};

// Test function to check DOM elements
window.testO11yElements = function() {
    console.log('Testing O11y DOM elements:');
    console.log('- o11y-sources-container:', document.getElementById('o11y-sources-container'));
    console.log('- o11y-sources-dropdown:', document.getElementById('o11y-sources-dropdown'));
    console.log('- o11y-sources-options:', document.getElementById('o11y-sources-options'));
    console.log('- o11y-sources-list:', document.getElementById('o11y-sources-list'));
    console.log('- o11y-sources-search:', document.getElementById('o11y-sources-search'));

    if (window.vuDataSimManager) {
        console.log('vuDataSimManager elements:', window.vuDataSimManager.elements.o11ySourcesList);
        console.log('O11y sources array:', window.vuDataSimManager.o11ySources.o11ySources);
        console.log('Selected sources:', window.vuDataSimManager.o11ySources.selectedO11ySources);
    }
};

// Export for potential module usage
if (typeof module !== 'undefined' && module.exports) {
    module.exports = VuDataSimManager;
}
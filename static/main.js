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
        this.logsManager.bindReloadButton(); // Bind the reload button after initialization
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

        // Load next test ID for display
        this.loadNextTestID();
        // Load next K6 test ID for display
        this.loadNextK6TestID();
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
            // K6 Load Testing elements
            k6TimeRange: document.getElementById('k6-time-range'),
            k6Vus: document.getElementById('k6-vus'),
            k6Iterations: document.getElementById('k6-iterations'),
            k6Interval: document.getElementById('k6-interval'),
            k6TestIdDisplay: document.getElementById('k6-test-id-display'),
            startK6TestBtn: document.getElementById('start-k6-test-btn'),
            logNodeFilter: document.getElementById('log-node'),
            logModuleFilter: document.getElementById('log-module'),
            logTypeFilter: document.getElementById('log-type'),
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
            o11ySourceMode: document.getElementById('o11y-source-mode'),
            o11yCategoryMode: document.getElementById('o11y-category-mode'),
            o11yCategorySelect: document.getElementById('o11y-category-select'),
            selectedCategoryInfo: document.getElementById('selected-category-info'),
            selectedCategoryName: document.getElementById('selected-category-name'),
            selectedCategorySources: document.getElementById('selected-category-sources'),
            epsSelect: document.getElementById('eps-select'),
            timeoutInput: document.getElementById('timeout-input'),
            skipChTruncate: document.getElementById('skip-ch-truncate'),
            startVuDataSimBtn: document.getElementById('start-vudatasim-btn'),
            stopVuDataSimBtn: document.getElementById('stop-vudatasim-btn'),
            syncProgressPercent: document.getElementById('sync-progress-percent'),
            syncProgressBar: document.getElementById('sync-progress-bar'),
            syncStatusText: document.getElementById('sync-status-text'),
            step1Icon: document.getElementById('step1-icon'),
            step1Text: document.getElementById('step1-text'),
            step1Title: document.getElementById('step1-title'),
            step1Container: document.getElementById('step1-container'),
            step1Box: document.getElementById('step1-box'),
            step2Icon: document.getElementById('step2-icon'),
            step2Text: document.getElementById('step2-text'),
            step2Title: document.getElementById('step2-title'),
            step2Container: document.getElementById('step2-container'),
            step2Box: document.getElementById('step2-box'),
            step3Icon: document.getElementById('step3-icon'),
            step3Text: document.getElementById('step3-text'),
            step3Title: document.getElementById('step3-title'),
            step3Container: document.getElementById('step3-container'),
            step3Box: document.getElementById('step3-box'),
            step4Icon: document.getElementById('step4-icon'),
            step4Text: document.getElementById('step4-text'),
            step4Title: document.getElementById('step4-title'),
            step4Container: document.getElementById('step4-container'),
            step4Box: document.getElementById('step4-box'),
            step5Icon: document.getElementById('step5-icon'),
            step5Text: document.getElementById('step5-text'),
            step5Title: document.getElementById('step5-title'),
            step5Container: document.getElementById('step5-container'),
            step5Box: document.getElementById('step5-box'),
            step6Icon: document.getElementById('step6-icon'),
            step6Text: document.getElementById('step6-text'),
            step6Title: document.getElementById('step6-title'),
            step6Container: document.getElementById('step6-container'),
            step6Box: document.getElementById('step6-box'),
            epsMode: document.getElementById('eps-mode'),
            epsCustomInput: document.getElementById('eps-custom-input'),
            testIdDisplay: document.getElementById('test-id-display'),


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

            // Bulk operation elements
            startAllBinariesBtn: document.getElementById('start-all-binaries-btn'),
            stopAllBinariesBtn: document.getElementById('stop-all-binaries-btn'),
            refreshAllStatusBtn: document.getElementById('refresh-all-status-btn'),
            bulkOperationLogs: document.getElementById('bulk-operation-logs'),
            bulkLogsOutput: document.getElementById('bulk-logs-output'),
            bulkOperationStatus: document.getElementById('bulk-operation-status'),
            bulkOperationProgress: document.getElementById('bulk-operation-progress'),

            // Dashboard elements
            nodeStatusIndicatorsContainer: document.getElementById('node-status-indicators'),

            // Chart value elements
            cpuMemoryValue: document.getElementById('cpu-memory-value'),

            // Kafka & ClickHouse cleaning elements
            // Add All Cluster Nodes button
            addAllClusterNodesBtn: document.getElementById('add-all-cluster-nodes-btn'),

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

            // Monitoring button
            monitoringBtn: document.getElementById('monitoring-btn'),

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

        // Add All Cluster Nodes button event listener
        this.elements.addAllClusterNodesBtn?.addEventListener('click', () => this.addAllClusterNodes());
        // vuDataSim control event listeners
        this.elements.startVuDataSimBtn?.addEventListener('click', () => this.startVuDataSim());
        this.elements.stopVuDataSimBtn?.addEventListener('click', () => this.stopVuDataSim());

        // New custom multi-select event listeners
        this.elements.o11ySourcesDropdown?.addEventListener('click', () => this.o11ySources.toggleO11ySourcesDropdown());
        this.elements.o11ySourcesSearch?.addEventListener('input', (e) => this.o11ySources.filterO11ySources(e.target.value));
        this.elements.o11ySourcesSelectAll?.addEventListener('click', () => this.o11ySources.selectAllO11ySources());
        this.elements.o11ySourcesClearAll?.addEventListener('click', () => this.o11ySources.clearAllO11ySources());

        // EPS Mode switching
        this.elements.epsMode?.addEventListener('change', () => {
            if (this.elements.epsMode.value === 'custom') {
                this.elements.epsSelect.classList.add('hidden');
                this.elements.epsCustomInput.classList.remove('hidden');
                this.elements.epsCustomInput.focus();
            } else {
                this.elements.epsSelect.classList.remove('hidden');
                this.elements.epsCustomInput.classList.add('hidden');
                this.elements.epsCustomInput.value = '';
            }
        });


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

        // Monitoring button event listeners
        console.log('Monitoring Button element:', this.elements.monitoringBtn);

        if (this.elements.monitoringBtn) {
            this.elements.monitoringBtn.addEventListener('click', () => {
                console.log('Monitoring button clicked!');
                window.location.href = '/static/monitoring.html';
            });
        } else {
            console.error('Monitoring button not found!');
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

        // Bulk binary control action listeners
        this.elements.startAllBinariesBtn?.addEventListener('click', () => this.startAllBinaries());
        this.elements.stopAllBinariesBtn?.addEventListener('click', () => this.stopAllBinaries());
        this.elements.refreshAllStatusBtn?.addEventListener('click', () => this.refreshAllBinaryStatus());

        // Add get binary output button listener
        const getOutputBtn = document.getElementById('get-binary-output-btn');
        if (getOutputBtn) {
            getOutputBtn.addEventListener('click', () => this.binaryControl.getBinaryOutput());
        }

        // K6 Load Testing event listeners
        if (this.elements.startK6TestBtn) {
            this.elements.startK6TestBtn.addEventListener('click', () => this.startK6Test());
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
        this.elements.logTypeFilter?.addEventListener('change', () => this.logsManager.filterLogs());

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

        // Binary status updates for button state - Update every 30 seconds
        setInterval(async () => {
            try {
                console.log('Periodic binary status check for button state...');
                await this.dashboard.updateVuDataSimButtonState();
            } catch (error) {
                console.error('Error in periodic binary status check:', error);
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

    async callAPI(endpoint, method = 'GET', data = null, timeoutMs = 30000) {
        const controller = new AbortController();
        let timeoutId;

        // Only set timeout if timeoutMs > 0
        if (timeoutMs > 0) {
            timeoutId = setTimeout(() => controller.abort(), timeoutMs);
        }

        try {
            const config = {
                method,
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: controller.signal,
            };

            if (data) {
                config.body = JSON.stringify(data);
            }

            const response = await fetch(`${this.apiBaseUrl}${endpoint}`, config);
            if (timeoutId) clearTimeout(timeoutId);

            // Get response data based on content type
            let responseData;
            const contentType = response.headers.get('content-type');

            if (contentType && contentType.includes('application/json')) {
                responseData = await response.json();
            } else {
                // For non-JSON responses (like HTML error pages), get as text
                responseData = await response.text();
            }

            // Check for HTTP error status codes and throw error for them
            if (!response.ok) {
                const errorMessage = (typeof responseData === 'object' && responseData.message) || responseData || `HTTP ${response.status}: ${response.statusText}`;
                const error = new Error(errorMessage);
                error.status = response.status;
                error.responseData = responseData;
                throw error;
            }

            // Return the actual response data for successful responses
            return responseData;
        } catch (error) {
            clearTimeout(timeoutId);
            console.error('API call failed:', error);

            if (error.name === 'AbortError') {
                throw new Error('Request timed out after 30 seconds');
            }

            // Only return mock response for actual network errors, not HTTP error responses or timeouts
            if (error.message.includes('fetch') || error.message.includes('network') || error.message.includes('CORS')) {
                console.warn('Network error detected, returning mock response:', error.message);
                return {
                    success: true,
                    message: 'Mock API response - Network error',
                    data: {}
                };
            }

            // For HTTP errors (including 500 errors) and timeouts, re-throw the error so calling code can handle it
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
        notification.className = `toast-message ${type}`;
        notification.textContent = message;

        // Add fade out animation before removing
        const removeNotification = () => {
            notification.style.animation = 'fadeOut 0.3s ease-out forwards';
            notification.addEventListener('animationend', () => {
                notification.remove();
            });
        };

        document.body.appendChild(notification);

        // Remove after 3 seconds
        setTimeout(() => {
            removeNotification();
        }, 3000);
    }


    async addAllClusterNodes() {
        try {
            // Disable the button and show loading state
            const button = this.elements.addAllClusterNodesBtn;
            const originalText = button.innerHTML;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Adding Nodes...</span>';
            button.disabled = true;

            console.log('Starting to add all cluster nodes...');

            // Call the API to add all cluster nodes (no timeout)
            const response = await this.callAPI('/api/nodes/add-all-cluster', 'POST', null, 0);

            if (!response.success) {
                throw new Error(response.message || 'Failed to add cluster nodes');
            }

            console.log('Successfully added cluster nodes:', response);

            // Show success notification
            this.showNotification(`Successfully added ${response.data.added_nodes.length} cluster nodes!`, 'success');

            // Refresh the dashboard to show new nodes
            await this.refreshDashboard();

        } catch (error) {
            console.error('Error adding cluster nodes:', error);
            this.showNotification(`Failed to add cluster nodes: ${error.message}`, 'error');

        } finally {
            // Re-enable the button and restore original text
            const button = this.elements.addAllClusterNodesBtn;
            button.innerHTML = '<span class="material-symbols-outlined">add_circle</span><span>Add All Cluster Nodes</span>';
            button.disabled = false;
        }
    }

    async startK6Test() {
        try {
            // Get values from input fields
            const timeRange = this.elements.k6TimeRange?.value?.trim() || '15m';
            const vus = parseInt(this.elements.k6Vus?.value) || 10;
            const iterations = parseInt(this.elements.k6Iterations?.value) || 5;
            const interval = parseInt(this.elements.k6Interval?.value) || 5;

            // Validate inputs
            if (!timeRange) {
                this.showNotification('Time range is required', 'error');
                return;
            }
            if (vus < 1 || vus > 1000) {
                this.showNotification('Virtual users must be between 1 and 1000', 'error');
                return;
            }
            if (iterations < 1 || iterations > 1000) {
                this.showNotification('Iterations must be between 1 and 1000', 'error');
                return;
            }
            if (interval < 1 || interval > 300) {
                this.showNotification('Interval must be between 1 and 300 seconds', 'error');
                return;
            }

            // Disable the button and show loading state
            const button = this.elements.startK6TestBtn;
            const originalText = button.innerHTML;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Running...</span>';
            button.disabled = true;

            console.log('Starting K6 test with parameters:', { timeRange, vus, iterations, interval });

            // Call the API to start K6 test (no timeout)
            const response = await this.callAPI('/api/k6/run-combined', 'POST', {
                timeRange,
                vus,
                iterations,
                interval
            }, 0);

            if (!response.success) {
                throw new Error(response.message || 'Failed to start K6 test');
            }

            console.log('K6 test started successfully:', response);

            // Update the K6 test ID display with the returned test ID
            if (response.data && response.data.test_id && this.elements.k6TestIdDisplay) {
                this.elements.k6TestIdDisplay.value = response.data.test_id;
            }

            // Load next K6 test ID for future use
            this.loadNextK6TestID();

            // Show success notification
            this.showNotification('K6 test was executed successfully!', 'success');

        } catch (error) {
            console.error('Error starting K6 test:', error);
            this.showNotification(`Failed to start K6 test: ${error.message}`, 'error');

        } finally {
            // Re-enable the button and restore original text
            const button = this.elements.startK6TestBtn;
            button.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start K6 Test</span>';
            button.disabled = false;
        }
    }

    async startVuDataSim() {
        try {
            // Get values from input fields
            const selectedSources = [...this.o11ySources.selectedO11ySources];
            // const selectedEPS = parseInt(this.elements.epsSelect.value);
            if (selectedSources.length === 0) {
                this.showNotification('Please select at least one o11y source', 'warning');
                return;
            }

            let selectedEPS;
            if (this.elements.epsMode.value === 'custom') {
                selectedEPS = parseInt(this.elements.epsCustomInput.value);
            } else{
                selectedEPS = parseInt(this.elements.epsSelect.value);
            }

            const timeoutInput = this.elements.timeoutInput?.value?.trim();
            const skipChTruncate = this.elements.skipChTruncate?.checked || false;
            let timeoutSeconds = 0;

            // Validate required timeout field
            if (!timeoutInput) {
                this.showNotification('Timeout is required', 'warning');
                return;
            }

            // Parse timeout
            timeoutSeconds = this.parseTimeoutToSeconds(timeoutInput);
            if (timeoutSeconds === null) {
                this.showNotification('Invalid timeout format. Use format like 60s, 5m, 3h or 1d ', 'error');
                return;
            }

            // Validate inputs


            if (!selectedEPS || selectedEPS <= 0) {
                console.log('Invalid EPS selected:', selectedEPS);
                this.showNotification('Please select a valid EPS target greater than 0', 'warning');
                return;
            }

            // Disable the start button and show loading state
            const startBtn = this.elements.startVuDataSimBtn;
            const stopBtn = this.elements.stopVuDataSimBtn;
            const originalStartText = startBtn.innerHTML;
            startBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Starting...</span>';
            startBtn.disabled = true;
            stopBtn.disabled = true;

            console.log('Starting vuDataSim with sources:', selectedSources, 'EPS:', selectedEPS, 'Timeout:', timeoutSeconds, 'Skip CH Truncate:', skipChTruncate);

            const sourceNameTranslation = {
                "LinuxMonitor": "Linux Monitor",
                "MongoDB": "MongoDB",
                "Mssql": "MSSQL",
                "Apache": "Apache",
                "Azure_Firewall": "Azure Firewall",
                "Azure_Redis_Cache": "Azure Redis Cache",
                "Traces": "Traces",
                "K8s": "Kubernetes"
            };
            const translatedSources = selectedSources.map(src => sourceNameTranslation[src] || src);

            // Create test run record first - belongs to the test run tracking process
            console.log('Creating test run record...');
            const testRunData = {
                target_eps: selectedEPS,
                o11y_sources: translatedSources,
                timeout_seconds: timeoutSeconds
            };

            const testRunResponse = await this.callAPI('/api/test-runs/start', 'POST', testRunData);
            if (!testRunResponse.success) {
                throw new Error(`Failed to create test run record: ${testRunResponse.message || 'Unknown error'}`);
            }

            const testId = testRunResponse.data.test_id;
            console.log('Test run created with ID:', testId);

            // Update the test ID display
            if (this.elements.testIdDisplay) {
                this.elements.testIdDisplay.value = testId;
            }

            // Store current test ID for stop operation
            this.currentTestId = testId;

            // Call the syncConfigs function (which now includes binary start)
            await this.o11ySources.syncConfigs(timeoutSeconds,selectedEPS, skipChTruncate);

            // Schedule pod deletion
            let podDeletionSuccess = false;
            try {
                const podDeletionResponse = await this.callAPI('/api/schedule-pod-deletion', 'POST', {
                    timeout_seconds: timeoutSeconds,
                    o11y_sources: selectedSources
                });
                console.log('Pod deletion scheduled:', podDeletionResponse);
                podDeletionSuccess = true;
            } catch (error) {
                console.error('Error scheduling pod deletion:', error);
                this.showNotification('Scheduled pod deletion failed', 'error');
                // Stop all binaries to cancel the running process
                try {
                    await this.callAPI('/api/binary/stop-all', 'POST');
                    console.log('All binaries stopped due to pod deletion failure');
                } catch (stopError) {
                    console.error('Error stopping binaries:', stopError);
                }
            }

            // Show success notification only if pod deletion succeeded
            if (podDeletionSuccess) {
                this.showNotification('vuDataSim started successfully!', 'success');
            }


            // // Show success notification only if syncConfigs completed without throwing an error
            // this.showNotification('vuDataSim started successfully!', 'success');

        } catch (error) {
            console.error('Error starting vuDataSim:', error);

            // If we created a test run record but sync failed, update status to failed - belongs to the test run tracking process
            if (this.currentTestId) {
                try {
                    console.log('Updating test run status to failed for ID:', this.currentTestId);
                    // Note: We'll need to add an API endpoint to update test run status
                    // For now, we'll just log it
                } catch (statusError) {
                    console.error('Error updating test run status:', statusError);
                }
            }

            this.showNotification(`Failed to start vuDataSim: ${error.message}`, 'error');

            // Re-enable buttons on error
            const startBtn = this.elements.startVuDataSimBtn;
            const stopBtn = this.elements.stopVuDataSimBtn;
            startBtn.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start vuDataSim</span>';
            startBtn.disabled = false;
            stopBtn.disabled = false;
        }
    }

    async stopVuDataSim() {
        try {
            // Disable buttons and show loading state
            const startBtn = this.elements.startVuDataSimBtn;
            const stopBtn = this.elements.stopVuDataSimBtn;
            const originalStopText = stopBtn.innerHTML;
            startBtn.disabled = true;
            stopBtn.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Stopping...</span>';
            stopBtn.disabled = true;

            console.log('Stopping vuDataSim...');

            // Call the API to stop all binaries
            const response = await this.callAPI('/api/binary/stop-all', 'POST');

            if (!response.success) {
                throw new Error(response.message || 'Failed to stop vuDataSim');
            }

            console.log('vuDataSim stopped successfully:', response);

            // Stop the test run record if we have a current test ID - belongs to the test run tracking process
            if (this.currentTestId) {
                console.log('Stopping test run record for ID:', this.currentTestId);
                try {
                    const stopTestRunResponse = await this.callAPI(`/api/test-runs/${this.currentTestId}/stop`, 'PUT');
                    if (stopTestRunResponse.success) {
                        console.log('Test run stopped successfully:', stopTestRunResponse.data);
                    } else {
                        console.warn('Failed to stop test run record:', stopTestRunResponse.message);
                    }
                } catch (stopError) {
                    console.error('Error stopping test run record:', stopError);
                    // Don't fail the entire operation for this
                }

                // Clear current test ID
                this.currentTestId = null;

                // Reload next test ID for display
                this.loadNextTestID();
            }

            // Update button states based on actual status from backend
            await this.dashboard.updateVuDataSimButtonState();

            // Show success notification
            this.showNotification('vuDataSim stopped successfully!', 'success');

        } catch (error) {
            console.error('Error stopping vuDataSim:', error);
            this.showNotification(`Failed to stop vuDataSim: ${error.message}`, 'error');

            // Update button states based on actual status from backend on error
            await this.dashboard.updateVuDataSimButtonState();
        }
    }

    parseTimeoutToSeconds(timeoutStr) {
        if (!timeoutStr || typeof timeoutStr !== 'string') {
            return null;
        }

        const match = timeoutStr.match(/^(\d+)([smhd])$/);
        if (!match) {
            return null;
        }

        const value = parseInt(match[1]);
        const unit = match[2];

        switch (unit) {
            case 's':
                return value;
            case 'm':
                return value * 60;
            case 'h':
                return value * 3600;
            case 'd':
                return value * 86400;
            default:
                return null;
        }
    }

    async startAllBinaries() {
        try {
            // Disable the button and show loading state
            const button = this.elements.startAllBinariesBtn;
            const originalText = button.innerHTML;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Starting All...</span>';
            button.disabled = true;

            // Show bulk operation logs
            this.showBulkOperationLogs();

            console.log('Starting all binaries on enabled nodes...');

            // Call the API to start all binaries
            const response = await this.callAPI('/api/binary/start-all', 'POST');

            if (!response.success) {
                throw new Error(response.message || 'Failed to start all binaries');
            }

            console.log('Bulk start operation completed:', response);

            // Update logs and progress
            this.updateBulkOperationLogs(response.data);

            // Show success notification
            this.showNotification(response.message, response.data.failed > 0 ? 'warning' : 'success');

            // Refresh all status after operation
            setTimeout(() => this.refreshAllBinaryStatus(), 2000);

        } catch (error) {
            console.error('Error starting all binaries:', error);
            this.showNotification(`Failed to start all binaries: ${error.message}`, 'error');

        } finally {
            // Re-enable the button and restore original text
            const button = this.elements.startAllBinariesBtn;
            button.innerHTML = '<span class="material-symbols-outlined">play_arrow</span><span>Start All Binaries</span>';
            button.disabled = false;
        }
    }

    async stopAllBinaries() {
        try {
            // Disable the button and show loading state
            const button = this.elements.stopAllBinariesBtn;
            const originalText = button.innerHTML;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">progress_activity</span><span>Stopping All...</span>';
            button.disabled = true;

            // Show bulk operation logs
            this.showBulkOperationLogs();

            console.log('Stopping all binaries on enabled nodes...');

            // Call the API to stop all binaries
            const response = await this.callAPI('/api/binary/stop-all', 'POST');

            if (!response.success) {
                throw new Error(response.message || 'Failed to stop all binaries');
            }

            console.log('Bulk stop operation completed:', response);

            // Update logs and progress
            this.updateBulkOperationLogs(response.data);

            // Show success notification
            this.showNotification(response.message, response.data.failed > 0 ? 'warning' : 'success');

            // Refresh all status after operation
            setTimeout(() => this.refreshAllBinaryStatus(), 2000);

        } catch (error) {
            console.error('Error stopping all binaries:', error);
            this.showNotification(`Failed to stop all binaries: ${error.message}`, 'error');

        } finally {
            // Re-enable the button and restore original text
            const button = this.elements.stopAllBinariesBtn;
            button.innerHTML = '<span class="material-symbols-outlined">stop</span><span>Stop All Binaries</span>';
            button.disabled = false;
        }
    }

    async refreshAllBinaryStatus() {
        try {
            console.log('Refreshing all binary statuses...');

            // Call the API to get all binary statuses
            const response = await this.callAPI('/api/binary/status', 'GET');

            if (!response.success) {
                throw new Error(response.message || 'Failed to refresh binary statuses');
            }

            console.log('Binary statuses refreshed:', response.data);

            // Update the binary control modal table
            this.binaryControl.updateAllStatusTable(response.data);

            // Show success notification
            this.showNotification(`Refreshed status for ${response.data.length} nodes`, 'success');

        } catch (error) {
            console.error('Error refreshing binary statuses:', error);
            this.showNotification(`Failed to refresh binary statuses: ${error.message}`, 'error');
        }
    }

    showBulkOperationLogs() {
        if (this.elements.bulkOperationLogs) {
            this.elements.bulkOperationLogs.classList.remove('hidden');
            if (this.elements.bulkLogsOutput) {
                this.elements.bulkLogsOutput.innerHTML = '<div class="text-text-secondary-light dark:text-text-secondary-dark">Operation starting...</div>';
            }
            if (this.elements.bulkOperationStatus) {
                this.elements.bulkOperationStatus.textContent = 'Operation in progress...';
            }
            if (this.elements.bulkOperationProgress) {
                this.elements.bulkOperationProgress.textContent = '0/0 completed';
            }
        }
    }

    updateBulkOperationLogs(data) {
        if (!data || !this.elements.bulkLogsOutput) return;

        const logsContainer = this.elements.bulkLogsOutput;
        logsContainer.innerHTML = '';

        // Display logs
        if (data.logs && data.logs.length > 0) {
            data.logs.forEach(log => {
                const logElement = document.createElement('div');
                logElement.className = 'text-xs mb-1';

                // Color code based on log content
                if (log.includes('SUCCESS:')) {
                    logElement.className += ' text-success';
                } else if (log.includes('FAILED:')) {
                    logElement.className += ' text-danger';
                } else {
                    logElement.className += ' text-text-secondary-light dark:text-text-secondary-dark';
                }

                logElement.textContent = log;
                logsContainer.appendChild(logElement);
            });
        }

        // Update status and progress
        if (this.elements.bulkOperationStatus) {
            this.elements.bulkOperationStatus.textContent = `Operation completed: ${data.successful}/${data.totalNodes} successful`;
        }
        if (this.elements.bulkOperationProgress) {
            this.elements.bulkOperationProgress.textContent = `${data.successful}/${data.totalNodes} completed`;
        }

        // Auto-scroll to bottom
        logsContainer.scrollTop = logsContainer.scrollHeight;
    }

    // Test run tracking functions - belongs to the test run tracking process
    async loadNextTestID() {
        try {
            console.log('Loading next test ID...');
            const response = await this.callAPI('/api/test-runs/next-id', 'GET');

            if (response.success && response.data && response.data.next_test_id) {
                if (this.elements.testIdDisplay) {
                    this.elements.testIdDisplay.value = response.data.next_test_id;
                }
                console.log('Next test ID loaded:', response.data.next_test_id);
            } else {
                console.warn('Failed to load next test ID:', response);
                // Generate a fallback UUID
                const fallbackUUID = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                    const r = Math.random() * 16 | 0;
                    const v = c === 'x' ? r : (r & 0x3 | 0x8);
                    return v.toString(16);
                });
                if (this.elements.testIdDisplay) {
                    this.elements.testIdDisplay.value = fallbackUUID;
                }
            }
        } catch (error) {
            console.error('Error loading next test ID:', error);
            // Generate a fallback UUID on error
            const fallbackUUID = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                const r = Math.random() * 16 | 0;
                const v = c === 'x' ? r : (r & 0x3 | 0x8);
                return v.toString(16);
            });
            if (this.elements.testIdDisplay) {
                this.elements.testIdDisplay.value = fallbackUUID;
            }
        }
    }

    // K6 test run tracking functions
    async loadNextK6TestID() {
        try {
            console.log('Loading next K6 test ID...');
            const response = await this.callAPI('/api/k6/next-test-id', 'GET');

            if (response.success && response.data && response.data.next_test_id) {
                if (this.elements.k6TestIdDisplay) {
                    this.elements.k6TestIdDisplay.value = response.data.next_test_id;
                }
                console.log('Next K6 test ID loaded:', response.data.next_test_id);
            } else {
                console.warn('Failed to load next K6 test ID:', response);
                // Generate a fallback UUID
                const fallbackUUID = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                    const r = Math.random() * 16 | 0;
                    const v = c === 'x' ? r : (r & 0x3 | 0x8);
                    return v.toString(16);
                });
                if (this.elements.k6TestIdDisplay) {
                    this.elements.k6TestIdDisplay.value = fallbackUUID;
                }
            }
        } catch (error) {
            console.error('Error loading next K6 test ID:', error);
            // Generate a fallback UUID on error
            const fallbackUUID = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                const r = Math.random() * 16 | 0;
                const v = c === 'x' ? r : (r & 0x3 | 0x8);
                return v.toString(16);
            });
            if (this.elements.k6TestIdDisplay) {
                this.elements.k6TestIdDisplay.value = fallbackUUID;
            }
        }
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
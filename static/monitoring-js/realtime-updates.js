// Real-time Updates Manager

class RealtimeUpdatesManager {
    constructor() {
        this.clusterManager = null;
        this.kafkaManager = null;
        this.podMetricsManager = null;
        this.systemHealthManager = null;
        this.isInitialized = false;
        this.updateInterval = 30000; // 30 seconds
        this.lastUpdate = null;
    }

    // Initialize all managers
    async initialize() {
        if (this.isInitialized) return;

        console.log('Initializing real-time monitoring...');

        try {
            // Initialize managers
            console.log('RealtimeUpdatesManager: Initializing managers...');
            this.clusterManager = new ClusterMetricsManager();
            this.kafkaManager = new KafkaMetricsManager();
            this.podMetricsManager = new PodMetricsManager();
            this.systemHealthManager = new SystemHealthManager();
            console.log('RealtimeUpdatesManager: Managers created');

            // Initialize each manager
            console.log('RealtimeUpdatesManager: Starting manager initialization...');
            await Promise.all([
                this.clusterManager.initialize(),
                this.kafkaManager.initialize(),
                this.podMetricsManager.initialize(),
                this.systemHealthManager.initialize()
            ]);
            console.log('RealtimeUpdatesManager: All managers initialized successfully');

            this.isInitialized = true;
            this.lastUpdate = new Date();

            console.log('Real-time monitoring initialized successfully');

            // Set up global refresh function
            window.refreshAllMonitoringData = () => this.refreshAll();

        } catch (error) {
            console.error('Failed to initialize real-time monitoring:', error);
        }
    }

    // Refresh all data
    async refreshAll() {
        if (!this.isInitialized) {
            await this.initialize();
            return;
        }

        console.log('Refreshing all monitoring data...');

        try {
            // Refresh all managers in parallel
            await Promise.all([
                this.clusterManager.refresh(),
                this.kafkaManager.refresh(),
                this.podMetricsManager.refresh(),
                this.systemHealthManager.refresh()
            ]);

            this.lastUpdate = new Date();
            this.updateLastUpdateTime();

            console.log('All monitoring data refreshed');
        } catch (error) {
            console.error('Error refreshing monitoring data:', error);
        }
    }

    // Update the last update timestamp in the UI
    updateLastUpdateTime() {
        const lastUpdateElement = document.getElementById('last-update');
        if (lastUpdateElement) {
            lastUpdateElement.textContent = this.lastUpdate.toLocaleTimeString();
        }
    }

    // Get current status of all managers
    getStatus() {
        return {
            isInitialized: this.isInitialized,
            lastUpdate: this.lastUpdate,
            managers: {
                cluster: this.clusterManager ? this.clusterManager.getMetrics() : null,
                kafka: this.kafkaManager ? this.kafkaManager.getMetrics() : null,
                pod: this.podMetricsManager ? this.podMetricsManager.getMetrics() : null,
                systemHealth: this.systemHealthManager ? this.systemHealthManager.getHealthData() : null
            }
        };
    }

    // Start periodic updates (called from main monitoring.js)
    startPeriodicUpdates() {
        if (!this.isInitialized) {
            this.initialize();
        }

        // Set up periodic refresh
        setInterval(() => {
            this.refreshAll();
        }, this.updateInterval);
    }

    // Handle section changes
    onSectionChange(sectionName) {
        if (!this.isInitialized) return;

        // Update data when switching to a section
        console.log('RealtimeUpdatesManager: Section changed to:', sectionName);
        console.log('RealtimeUpdatesManager: podMetricsManager available:', !!this.podMetricsManager);

        switch (sectionName) {
            case 'overview':
                console.log('RealtimeUpdatesManager: Refreshing cluster manager for overview');
                // Overview uses cluster metrics
                this.clusterManager.refresh();
                break;
            case 'performance':
                console.log('RealtimeUpdatesManager: Refreshing kafka manager for performance');
                // Performance section uses Kafka metrics
                this.kafkaManager.refresh();
                break;
            case 'pod-metrics':
                console.log('RealtimeUpdatesManager: Refreshing pod metrics manager');
                // Pod metrics section uses pod metrics
                if (this.podMetricsManager) {
                    this.podMetricsManager.refresh();
                } else {
                    console.error('RealtimeUpdatesManager: podMetricsManager not available for refresh');
                }
                break;
            case 'system':
                console.log('RealtimeUpdatesManager: Refreshing system health manager');
                // System section uses system health and pod metrics
                this.systemHealthManager.refresh();
                break;
        }
    }

    // Handle visibility changes (pause/resume updates when tab is not visible)
    handleVisibilityChange() {
        if (document.hidden) {
            console.log('Tab hidden, pausing updates');
            // In a real implementation, you might want to pause updates
        } else {
            console.log('Tab visible, resuming updates');
            // Refresh data when tab becomes visible
            if (this.isInitialized) {
                this.refreshAll();
            }
        }
    }

    // Clean up resources
    destroy() {
        if (this.clusterManager) {
            // Clean up cluster manager if needed
        }
        if (this.kafkaManager) {
            // Clean up kafka manager if needed
        }
        if (this.systemHealthManager) {
            // Clean up system health manager if needed
        }

        this.isInitialized = false;
        console.log('Real-time monitoring destroyed');
    }
}

// Export for use in other modules
window.RealtimeUpdatesManager = RealtimeUpdatesManager;
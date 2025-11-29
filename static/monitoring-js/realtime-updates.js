// Real-time Updates Manager

class RealtimeUpdatesManager {
    constructor() {
        this.clusterManager = null;
        this.kafkaManager = null;
        this.systemHealthManager = null;
        this.isInitialized = false;
        this.updateInterval = 30000; // 30 seconds
        this.lastUpdate = null;
        this.updateIntervalId = null; // Track interval ID for cleanup
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
            this.systemHealthManager = new SystemHealthManager();
            console.log('RealtimeUpdatesManager: Managers created');

            // Initialize each manager
            console.log('RealtimeUpdatesManager: Starting manager initialization...');
            await Promise.all([
                this.clusterManager.initialize(),
                this.kafkaManager.initialize(),
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
                systemHealth: this.systemHealthManager ? this.systemHealthManager.getHealthData() : null
            }
        };
    }

    // Start periodic updates (called from main monitoring.js)
    startPeriodicUpdates() {
        if (!this.isInitialized) {
            this.initialize();
        }

        // Clear any existing interval
        if (this.updateIntervalId) {
            clearInterval(this.updateIntervalId);
            this.updateIntervalId = null;
        }

        // Set up periodic refresh
        this.updateIntervalId = setInterval(() => {
            this.refreshAll();
        }, this.updateInterval);
    }

    // Handle section changes
    onSectionChange(sectionName) {
        if (!this.isInitialized) return;

        // Update data when switching to a section

        switch (sectionName) {
            case 'overview':
                console.log('RealtimeUpdatesManager: Refreshing cluster manager for overview');
                // Overview uses cluster metrics
                this.clusterManager.refresh();
                break;
            case 'performance':
                console.log('RealtimeUpdatesManager: Refreshing kafka manager for performance');
                // Performance section uses Kafka metrics
                this.kafkaManager.onSectionVisible();
                this.kafkaManager.refresh();
                break;
            case 'system':
                console.log('RealtimeUpdatesManager: Refreshing system health manager');
                // System section uses system health
                this.systemHealthManager.refresh();
                break;
        }
    }

    // Handle visibility changes (pause/resume updates when tab is not visible)
    handleVisibilityChange() {
        if (document.hidden) {
            console.log('Tab hidden, pausing updates');
            // Actually pause updates to save resources
            if (this.updateIntervalId) {
                clearInterval(this.updateIntervalId);
                this.updateIntervalId = null;
            }
        } else {
            console.log('Tab visible, resuming updates');
            // Refresh data immediately when tab becomes visible
            if (this.isInitialized) {
                this.refreshAll();
                // Restart periodic updates
                this.startPeriodicUpdates();
            }
        }
    }

    // Clean up resources
    destroy() {
        // Clear periodic update interval
        if (this.updateIntervalId) {
            clearInterval(this.updateIntervalId);
            this.updateIntervalId = null;
        }

        // Clean up child managers
        if (this.clusterManager && typeof this.clusterManager.destroy === 'function') {
            this.clusterManager.destroy();
        }
        if (this.kafkaManager && typeof this.kafkaManager.destroy === 'function') {
            this.kafkaManager.destroy();
        }
        if (this.systemHealthManager && typeof this.systemHealthManager.destroy === 'function') {
            this.systemHealthManager.destroy();
        }

        this.clusterManager = null;
        this.kafkaManager = null;
        this.systemHealthManager = null;
        this.isInitialized = false;

        console.log('Real-time monitoring destroyed and cleaned up');
    }
}

// Export for use in other modules
window.RealtimeUpdatesManager = RealtimeUpdatesManager;
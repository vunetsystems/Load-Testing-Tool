// Monitoring Dashboard JavaScript

// Global realtime updates manager instance
let realtimeManager = null;

document.addEventListener('DOMContentLoaded', function() {
    // Initialize monitoring page
    initializeMonitoring();

    // Set up navigation
    setupNavigation();

    // Set up real-time updates
    setupRealTimeUpdates();
});

async function initializeMonitoring() {
    console.log('initializeMonitoring: Starting monitoring dashboard initialization...');

    // Initialize the real-time updates manager
    console.log('initializeMonitoring: Creating RealtimeUpdatesManager...');
    realtimeManager = new RealtimeUpdatesManager();
    console.log('initializeMonitoring: RealtimeUpdatesManager created, calling initialize...');
    await realtimeManager.initialize();
    console.log('initializeMonitoring: RealtimeUpdatesManager initialized');

    // Update last update timestamp
    updateLastUpdateTime();

    // Set up section navigation
    setupSectionNavigation();

    // Initialize any charts or visualizations
    initializeCharts();

    console.log('initializeMonitoring: Monitoring dashboard initialized successfully');
}

function setupNavigation() {
    // Handle back to dashboard button
    const backButton = document.getElementById('back-to-dashboard-btn');
    if (backButton) {
        backButton.addEventListener('click', function() {
            window.location.href = '/static/index.html';
        });
    }

    // Handle refresh button
    const refreshButton = document.getElementById('refresh-monitoring-btn');
    if (refreshButton) {
        refreshButton.addEventListener('click', function() {
            refreshMonitoringData();
        });
    }
}

function setupSectionNavigation() {
    // Handle monitoring section navigation
    const navButtons = document.querySelectorAll('.monitoring-nav');
    navButtons.forEach(button => {
        button.addEventListener('click', function() {
            const section = this.getAttribute('data-section');
            showSection(section);

            // Update active state
            navButtons.forEach(btn => btn.classList.remove('active'));
            this.classList.add('active');

            // Handle tab initialization for performance section
            if (section === 'performance') {
                setupPerformanceTabs();
            }

            // Notify realtime manager of section change
            if (realtimeManager) {
                realtimeManager.onSectionChange(section);
            }
        });
    });
}

function showSection(sectionName) {
    // Hide all sections
    const sections = document.querySelectorAll('#monitoring-content > div');
    sections.forEach(section => {
        section.classList.add('hidden');
    });

    // Show selected section
    const targetSection = document.getElementById(sectionName + '-section');
    if (targetSection) {
        targetSection.classList.remove('hidden');
    }

    // Update page title
    updatePageTitle(sectionName);
}

function updatePageTitle(sectionName) {
    const titleElement = document.getElementById('page-title');
    if (titleElement) {
        const titles = {
            'overview': 'Monitoring Overview',
            'performance': 'Kafka Metrics',
            'pod-metrics': 'Pod Metrics',
            'system': 'System Health',
            'logs': 'Log Analysis',
            'alerts': 'Alerts & Notifications'
        };
        titleElement.textContent = titles[sectionName] || 'Monitoring Dashboard';
    }
}

function setupRealTimeUpdates() {
    // Set up visibility change handling
    document.addEventListener('visibilitychange', function() {
        if (realtimeManager) {
            realtimeManager.handleVisibilityChange();
        }
    });

    // Start periodic updates if manager is available
    if (realtimeManager) {
        realtimeManager.startPeriodicUpdates();
    }
}

function updateLastUpdateTime() {
    const lastUpdateElement = document.getElementById('last-update');
    if (lastUpdateElement) {
        const now = new Date();
        lastUpdateElement.textContent = now.toLocaleTimeString();
    }
}

async function refreshMonitoringData() {
    // Show loading state
    showLoadingState();

    try {
        // Refresh all data using the realtime manager
        if (realtimeManager) {
            await realtimeManager.refreshAll();
        } else {
            // Fallback to direct refresh if manager not available
            await refreshAllData();
        }
    } catch (error) {
        console.error('Error refreshing monitoring data:', error);
    } finally {
        // Hide loading state
        hideLoadingState();
    }
}

// Fallback refresh function for direct use
async function refreshAllData() {
    if (window.clusterManager) {
        await window.clusterManager.refresh();
    }
    if (window.kafkaManager) {
        await window.kafkaManager.refresh();
    }
    if (window.podMetricsManager) {
        await window.podMetricsManager.refresh();
    }
    if (window.systemHealthManager) {
        await window.systemHealthManager.refresh();
    }
}

function showLoadingState() {
    // Add loading indicators to key elements
    const buttons = document.querySelectorAll('button');
    buttons.forEach(button => {
        if (button.id === 'refresh-monitoring-btn') {
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">refresh</span><span>Refreshing...</span>';
            button.disabled = true;
        }
    });
}

function hideLoadingState() {
    // Remove loading indicators
    const buttons = document.querySelectorAll('button');
    buttons.forEach(button => {
        if (button.id === 'refresh-monitoring-btn') {
            button.innerHTML = '<span class="material-symbols-outlined">refresh</span><span>Refresh Data</span>';
            button.disabled = false;
        }
    });
}

function setupPerformanceTabs() {
    // Set up tab switching for performance section
    const tabButtons = document.querySelectorAll('.performance-tab');
    tabButtons.forEach(button => {
        button.addEventListener('click', function() {
            const tabName = this.getAttribute('data-tab');

            // Update active tab button
            tabButtons.forEach(btn => {
                btn.classList.remove('active', 'border-primary', 'dark:border-primary-dark', 'text-primary', 'dark:text-primary-dark');
                btn.classList.add('border-transparent', 'text-text-secondary-light', 'dark:text-text-secondary-dark');
            });

            this.classList.add('active', 'border-primary', 'dark:border-primary-dark', 'text-primary', 'dark:text-primary-dark');
            this.classList.remove('border-transparent', 'text-text-secondary-light', 'dark:text-text-secondary-dark');

            // Show corresponding tab content
            showPerformanceTab(tabName);
        });
    });

    // Initialize with first tab (Kafka metrics)
    if (tabButtons.length > 0) {
        tabButtons[0].click();
    }
}

function showPerformanceTab(tabName) {
    // Hide all tab contents
    const tabContents = document.querySelectorAll('.performance-tab-content');
    tabContents.forEach(content => {
        content.classList.add('hidden');
    });

    // Show selected tab content
    const targetTab = document.getElementById(tabName + '-tab');
    if (targetTab) {
        targetTab.classList.remove('hidden');
    }

    // Trigger specific data loading for the selected tab
    if (realtimeManager) {
        console.log('showPerformanceTab: Switching to tab:', tabName);
        console.log('showPerformanceTab: realtimeManager available:', !!realtimeManager);

        switch (tabName) {
            case 'kafka':
                console.log('showPerformanceTab: Refreshing kafka manager');
                if (realtimeManager.kafkaManager) {
                    realtimeManager.kafkaManager.refresh();
                }
                break;
            case 'pod':
                console.log('showPerformanceTab: Refreshing pod metrics manager');
                console.log('showPerformanceTab: podMetricsManager available:', !!realtimeManager.podMetricsManager);
                if (realtimeManager.podMetricsManager) {
                    realtimeManager.podMetricsManager.refresh();
                } else {
                    console.error('showPerformanceTab: podMetricsManager not available');
                }
                break;
            case 'cluster':
                console.log('showPerformanceTab: Refreshing cluster manager');
                if (realtimeManager.clusterManager) {
                    realtimeManager.clusterManager.refresh();
                }
                break;
            case 'system':
                console.log('showPerformanceTab: Refreshing system health manager');
                if (realtimeManager.systemHealthManager) {
                    realtimeManager.systemHealthManager.refresh();
                }
                break;
        }
    }
}

function initializeCharts() {
    // Placeholder for chart initialization
    // In a real application, you would initialize Chart.js, D3.js, or other charting libraries here
    console.log('Charts initialized');
}

// Handle monitoring button click from main dashboard (runs once on page load)
const monitoringButton = document.getElementById('monitoring-btn');
if (monitoringButton) {
    monitoringButton.addEventListener('click', function() {
        window.location.href = 'http://164.52.213.158:8086/static/monitoring.html';
    });
}

// Global function to get monitoring status (for debugging)
window.getMonitoringStatus = function() {
    if (realtimeManager) {
        return realtimeManager.getStatus();
    }
    return { status: 'not_initialized' };
};
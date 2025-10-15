// Metrics Management for vuDataSim
class MetricsManager {
    constructor(vuDataSimManager) {
        this.manager = vuDataSimManager;
        this.timePicker = null;
        this.initializeTimePicker();
        this.bindEvents();
    }

    initializeTimePicker() {
        const timePickerElement = document.getElementById('metrics-time-range');
        if (!timePickerElement) return;

        this.timePicker = flatpickr(timePickerElement, {
            mode: 'range',
            enableTime: true,
            dateFormat: 'Y-m-d H:i',
            defaultDate: [
                new Date(Date.now() - 5 * 60 * 1000), // 5 minutes ago
                new Date()
            ],
            onChange: (selectedDates) => {
                if (selectedDates.length === 2) {
                    this.updateMetrics(selectedDates[0], selectedDates[1]);
                }
            }
        });
    }

    bindEvents() {
        const refreshButton = document.getElementById('refresh-clickhouse-metrics-btn');
        if (refreshButton) {
            refreshButton.addEventListener('click', () => {
                const dates = this.timePicker.selectedDates;
                if (dates.length === 2) {
                    this.updateMetrics(dates[0], dates[1]);
                } else {
                    // Default to last 5 minutes if no range selected
                    const end = new Date();
                    const start = new Date(end - 5 * 60 * 1000);
                    this.updateMetrics(start, end);
                }
            });
        }

        // Add preset button event listeners
        const preset1Min = document.getElementById('preset-1min');
        const preset5Min = document.getElementById('preset-5min');
        const preset15Min = document.getElementById('preset-15min');

        if (preset1Min) {
            preset1Min.addEventListener('click', () => this.setPresetRange(1));
        }
        if (preset5Min) {
            preset5Min.addEventListener('click', () => this.setPresetRange(5));
        }
        if (preset15Min) {
            preset15Min.addEventListener('click', () => this.setPresetRange(15));
        }
    }

    async fetchMetrics(startTime, endTime) {
        try {
            const start = startTime.toISOString();
            const end = endTime.toISOString();
            const response = await fetch(`${this.manager.apiBaseUrl}/api/metrics?start=${start}&end=${end}`);
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const data = await response.json();
            if (!data.success) {
                throw new Error(data.message || 'Failed to fetch metrics');
            }
            
            return data.data;
        } catch (error) {
            console.error('Error fetching metrics:', error);
            throw error;
        }
    }

    async updateMetrics(startTime = null, endTime = null) {
        try {
            // Default to last 5 minutes if no time range provided
            const end = endTime || new Date();
            const start = startTime || new Date(end - 5 * 60 * 1000);

            const metrics = await this.fetchMetrics(start, end);
            this.manager.displayClickHouseMetrics(metrics);
        } catch (error) {
            console.error('Error updating metrics:', error);
            // TODO: Show error in UI
        }
    }

    setPresetRange(minutes) {
        const end = new Date();
        const start = new Date(end - minutes * 60 * 1000);

        // Update the time picker display
        this.timePicker.setDate([start, end]);

        // Update metrics immediately
        this.updateMetrics(start, end);
    }
}

// Logs Management Module
class LogsManager {
    constructor(manager) {
        this.manager = manager;
        this.logEntries = [];
        this.lastLogUpdate = 0;
    }

    bindReloadButton() {
        const reloadBtn = document.getElementById('reload-logs-btn');
        if (reloadBtn) {
            reloadBtn.addEventListener('click', () => this.clearLogs());
        } else {
            console.error('Reload logs button not found');
        }
    }

    async loadLogs() {
        try {
            const response = await this.manager.callAPI('/api/logs?limit=50');
            if (response.success && response.data && response.data.logs) {
                this.logEntries = response.data.logs.map(log => ({
                    time: log.timestamp,
                    node: log.node,
                    module: log.module,
                    message: log.message,
                    type: log.type
                }));
                this.displayLogs(this.logEntries);
            }
        } catch (error) {
            console.error('Error loading logs:', error);
        }
    }

    populateNodeFilters() {
        // Populate log filter dropdown with actual node names
        const nodeFilter = this.manager.elements.logNodeFilter;
        if (!nodeFilter) return;

        // Clear existing options except "All Nodes"
        while (nodeFilter.children.length > 1) {
            nodeFilter.removeChild(nodeFilter.lastChild);
        }

        // Add actual node names (include both active and error status nodes)
        Object.keys(this.manager.nodeData).forEach(nodeId => {
            const option = document.createElement('option');
            option.value = nodeId;
            option.textContent = nodeId;
            nodeFilter.appendChild(option);
        });
    }

    filterLogs() {
        const nodeFilter = this.manager.elements.logNodeFilter.value;
        const moduleFilter = this.manager.elements.logModuleFilter.value;

        const filteredLogs = this.logEntries.filter(log => {
            const nodeMatch = nodeFilter === 'All Nodes' || log.node === nodeFilter;
            const moduleMatch = moduleFilter === 'All Modules' || log.module === moduleFilter;
            return nodeMatch && moduleMatch;
        });

        this.displayLogs(filteredLogs);
    }

    displayLogs(logs) {
        const container = this.manager.elements.logsContainer;
        container.innerHTML = '';

        logs.forEach((log, index) => {
            const logElement = document.createElement('p');
            logElement.className = 'animate-fade-in';
            logElement.style.animationDelay = `${index * 50}ms`;

            const typeClass = this.getLogTypeClass(log.type);
            logElement.innerHTML = `
                <span class="text-sky-400">${log.time}</span> -
                <span class="text-purple-400">${log.node}</span> -
                <span class="${typeClass}">${log.module}</span>: ${log.message}
            `;

            container.appendChild(logElement);
        });
    }

    clearLogs() {
        this.logEntries = [];
        this.displayLogs([]);
    }

    getLogTypeClass(type) {
        const classes = {
            info: 'text-emerald-400',
            warning: 'text-yellow-400',
            error: 'text-red-400',
            success: 'text-emerald-400',
            metric: 'text-emerald-400'
        };
        return classes[type] || 'text-gray-400';
    }

    addRandomLog() {
        const randomMessages = [
            'Processing batch request...',
            'Cache updated successfully',
            'Connection pool optimized',
            'Load balancer adjusted',
            'Memory usage optimized',
            'Network latency stable',
            'Database query optimized'
        ];

        const randomMessage = randomMessages[Math.floor(Math.random() * randomMessages.length)];
        const currentTime = new Date().toISOString().slice(0, 19).replace('T', ' ');
        const activeNodes = Object.keys(this.manager.nodeData).filter(id => this.manager.nodeData[id].status === 'active' || this.manager.nodeData[id].status === 'error');
        const randomNode = activeNodes[Math.floor(Math.random() * activeNodes.length)];

        const newLog = {
            time: currentTime,
            node: randomNode.charAt(0).toUpperCase() + randomNode.slice(1),
            module: 'Module A',
            message: randomMessage,
            type: 'info'
        };

        this.logEntries.unshift(newLog);
        if (this.logEntries.length > 50) {
            this.logEntries.pop(); // Keep only latest 50 logs
        }

        this.filterLogs(); // Refresh the display
    }
}
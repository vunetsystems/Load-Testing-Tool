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
                this.logEntries = response.data.logs.map(logItem => {
                    let log;
                    if (typeof logItem === 'string') {
                        log = JSON.parse(logItem);
                    } else {
                        log = logItem;
                    }
                    return {
                        time: log.timestamp,
                        node: log.node,
                        module: log.module,
                        message: log.message,
                        type: log.level || log.type // Use 'level' from JSON, fallback to 'type'
                    };
                });
                this.populateNodeFilters(); // Populate filters after loading logs
                this.filterLogs(); // Apply current filters instead of displaying all logs
            }
        } catch (error) {
            console.error('Error loading logs:', error);
        }
    }

    populateNodeFilters() {
        // Populate log filter dropdowns with actual values from logs
        const nodeFilter = this.manager.elements.logNodeFilter;
        const moduleFilter = this.manager.elements.logModuleFilter;
        const typeFilter = this.manager.elements.logTypeFilter;

        if (!nodeFilter || !moduleFilter || !typeFilter) return;

        // Store current selections before clearing
        const currentNodeValue = nodeFilter.value;
        const currentModuleValue = moduleFilter.value;
        const currentTypeValue = typeFilter.value;

        // Clear existing options except "All Nodes/Modules/Types"
        while (nodeFilter.children.length > 1) {
            nodeFilter.removeChild(nodeFilter.lastChild);
        }
        while (moduleFilter.children.length > 1) {
            moduleFilter.removeChild(moduleFilter.lastChild);
        }
        while (typeFilter.children.length > 1) {
            typeFilter.removeChild(typeFilter.lastChild);
        }

        // Collect unique values from logs
        const uniqueNodes = new Set();
        const uniqueModules = new Set();
        const uniqueTypes = new Set();

        this.logEntries.forEach(log => {
            if (log.node) uniqueNodes.add(log.node);
            if (log.module) uniqueModules.add(log.module);
            if (log.type) uniqueTypes.add(log.type);
        });

        // Add node options
        uniqueNodes.forEach(node => {
            const option = document.createElement('option');
            option.value = node;
            option.textContent = node;
            nodeFilter.appendChild(option);
        });

        // Add module options
        uniqueModules.forEach(module => {
            const option = document.createElement('option');
            option.value = module;
            option.textContent = module;
            moduleFilter.appendChild(option);
        });

        // Add type options (capitalize first letter)
        uniqueTypes.forEach(type => {
            const option = document.createElement('option');
            option.value = type;
            option.textContent = type.charAt(0).toUpperCase() + type.slice(1);
            typeFilter.appendChild(option);
        });

        // Restore previous selections if they still exist in the options
        if (currentNodeValue && Array.from(nodeFilter.options).some(opt => opt.value === currentNodeValue)) {
            nodeFilter.value = currentNodeValue;
        }
        if (currentModuleValue && Array.from(moduleFilter.options).some(opt => opt.value === currentModuleValue)) {
            moduleFilter.value = currentModuleValue;
        }
        if (currentTypeValue && Array.from(typeFilter.options).some(opt => opt.value === currentTypeValue)) {
            typeFilter.value = currentTypeValue;
        }
    }

    filterLogs() {
        const nodeFilter = this.manager.elements.logNodeFilter.value;
        const moduleFilter = this.manager.elements.logModuleFilter.value;
        const typeFilter = this.manager.elements.logTypeFilter.value;

        const filteredLogs = this.logEntries.filter(log => {
            const nodeMatch = nodeFilter === 'All Nodes' || log.node === nodeFilter;
            const moduleMatch = moduleFilter === 'All Modules' || log.module === moduleFilter;
            const typeMatch = typeFilter === 'All Types' || log.type.toLowerCase() === typeFilter.toLowerCase();
            return nodeMatch && moduleMatch && typeMatch;
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
            metric: 'text-emerald-400',
            debug: 'text-blue-400',
            fatal: 'text-red-500'
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
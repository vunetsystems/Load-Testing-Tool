// Node Management Module
class NodeManagement {
    constructor(manager) {
        this.manager = manager;
        this.isEditMode = false;
        this.editNodeName = null;
    }

    // Legacy function for backward compatibility (browser cache)
    toggleNodeManagement() {
        console.log('toggleNodeManagement called - redirecting to modal');
        console.log('Modal element available:', !!this.manager.elements.nodeManagementModal);
        console.log('Modal element ID:', this.manager.elements.nodeManagementModal?.id);
        this.openNodeManagementModal();
    }

    openNodeManagementModal() {
        console.log('Opening node management modal');
        console.log('Available elements:', Object.keys(this.manager.elements));
        console.log('Modal element search:', document.getElementById('node-management-modal'));
        const modal = this.manager.elements.nodeManagementModal;

        if (!modal) {
            console.error('Node management modal not found!');
            console.error('Current modal element:', modal);
            console.error('Direct DOM query:', document.getElementById('node-management-modal'));
            return;
        }

        modal.classList.remove('hidden');
        // Refresh nodes table when modal opens
        this.refreshNodesTable();
    }

    closeNodeManagementModal() {
        console.log('Closing node management modal');
        const modal = this.manager.elements.nodeManagementModal;

        if (!modal) {
            console.error('Node management modal not found!');
            return;
        }

        modal.classList.add('hidden');
        // Clear form when closing modal
        this.clearNodeForm();
    }

    async addNode() {
        // If in edit mode, call update instead
        if (this.isEditMode) {
            await this.updateNode();
            return;
        }

        const nodeData = {
            host: this.manager.elements.nodeHost.value,
            user: this.manager.elements.nodeUser.value,
            key_path: this.manager.elements.nodeKeypath.value,
            conf_dir: this.manager.elements.nodeConfdir.value,
            binary_dir: this.manager.elements.nodeBindir.value,
            description: this.manager.elements.nodeDescription.value,
            enabled: this.manager.elements.nodeEnabled.checked
        };

        if (!nodeData.host || !nodeData.user || !nodeData.key_path || !nodeData.conf_dir || !nodeData.binary_dir) {
            this.manager.showNotification('Please fill in all required fields', 'error');
            return;
        }

        this.setButtonLoading(this.manager.elements.addNodeBtn, true);

        try {
            const nodeName = this.manager.elements.nodeName.value || `node-${Date.now()}`;
            const response = await this.manager.callAPI(`/api/nodes/${nodeName}`, 'POST', nodeData);

            if (response.success) {
                this.manager.showNotification(`Node ${nodeName} added successfully`, 'success');
                this.clearNodeForm();
                this.refreshNodesTable();
                this.manager.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to add node');
            }
        } catch (error) {
            console.error('Error adding node:', error);
            this.manager.showNotification('Failed to add node: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.manager.elements.addNodeBtn, false);
        }
    }

    clearNodeForm() {
        this.manager.elements.nodeName.value = '';
        this.manager.elements.nodeHost.value = '';
        this.manager.elements.nodeUser.value = '';
        this.manager.elements.nodeKeypath.value = '';
        this.manager.elements.nodeConfdir.value = '';
        this.manager.elements.nodeBindir.value = '';
        this.manager.elements.nodeDescription.value = '';
        this.manager.elements.nodeEnabled.checked = true;

        // Exit edit mode if active
        if (this.isEditMode) {
            this.exitEditMode();
        }
    }

    async refreshNodesTable() {
        try {
            const response = await this.manager.callAPI('/api/nodes');
            if (response.success && response.data) {
                this.displayNodesTable(response.data);
            }
        } catch (error) {
            console.error('Error refreshing nodes table:', error);
        }
    }

    displayNodesTable(nodes) {
        const tbody = this.manager.elements.nodesTableBody;
        tbody.innerHTML = '';

        if (nodes.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="5" class="p-4 text-center text-text-secondary-light dark:text-text-secondary-dark">No nodes configured</td>';
            tbody.appendChild(row);
            return;
        }

        nodes.forEach(node => {
            const row = document.createElement('tr');
            row.className = 'hover:bg-subtle-light/50 dark:hover:bg-subtle-dark/50 transition-colors duration-200';

            const statusBadge = node.enabled
                ? '<div class="inline-flex items-center gap-2 rounded-full bg-success/20 dark:bg-success-dark/20 px-3 py-1 text-xs font-medium text-success dark:text-success-dark"><span class="h-2 w-2 rounded-full bg-success"></span>Enabled</div>'
                : '<div class="inline-flex items-center gap-2 rounded-full bg-danger/20 dark:bg-danger-dark/20 px-3 py-1 text-xs font-medium text-danger dark:text-danger-dark"><span class="h-2 w-2 rounded-full bg-danger"></span>Disabled</div>';

            row.innerHTML = `
                <td class="p-4 font-medium">${node.name}</td>
                <td class="p-4">${node.host}</td>
                <td class="p-4">${statusBadge}</td>
                <td class="p-4">${node.description || '-'}</td>
                <td class="p-4">
                    <div class="flex items-center gap-2">
                        <button onclick="window.vuDataSimManager.nodeManagement.editNode('${node.name}')" class="px-3 py-1 text-xs rounded bg-primary/20 text-primary hover:bg-primary/30 transition-colors">
                            <span class="material-symbols-outlined text-sm mr-1">edit</span>
                            Edit
                        </button>
                        <button onclick="window.vuDataSimManager.nodeManagement.toggleNode('${node.name}', ${!node.enabled})" class="px-3 py-1 text-xs rounded ${node.enabled ? 'bg-danger/20 text-danger hover:bg-danger/30' : 'bg-success/20 text-success hover:bg-success/30'} transition-colors">
                            ${node.enabled ? 'Disable' : 'Enable'}
                        </button>
                        <button onclick="window.vuDataSimManager.nodeManagement.removeNode('${node.name}')" class="px-3 py-1 text-xs rounded bg-danger/20 text-danger hover:bg-danger/30 transition-colors">
                            Remove
                        </button>
                    </div>
                </td>
            `;

            tbody.appendChild(row);
        });
    }

    async toggleNode(nodeName, enable) {
        try {
            const response = await this.manager.callAPI(`/api/nodes/${nodeName}`, 'PUT', { enabled: enable });

            if (response.success) {
                this.manager.showNotification(`Node ${nodeName} ${enable ? 'enabled' : 'disabled'} successfully`, 'success');
                this.refreshNodesTable();
                this.manager.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to update node');
            }
        } catch (error) {
            console.error('Error toggling node:', error);
            this.manager.showNotification('Failed to update node: ' + error.message, 'error');
        }
    }

    async removeNode(nodeName) {
        if (!confirm(`Are you sure you want to remove node "${nodeName}"?`)) {
            return;
        }

        try {
            const response = await this.manager.callAPI(`/api/nodes/${nodeName}`, 'DELETE');

            if (response.success) {
                this.manager.showNotification(`Node ${nodeName} removed successfully`, 'success');
                this.refreshNodesTable();
                this.manager.loadNodes(); // Refresh the dashboard nodes too
            } else {
                throw new Error(response.message || 'Failed to remove node');
            }
        } catch (error) {
            console.error('Error removing node:', error);
            this.manager.showNotification('Failed to remove node: ' + error.message, 'error');
        }
    }

    editNode(nodeName) {
        console.log('Editing node:', nodeName);
        // For now, we'll use the same form but in edit mode
        // In a real implementation, you might want to pre-populate the form with existing data
        this.isEditMode = true;
        this.editNodeName = nodeName;

        // Update the form title and button text
        const formTitle = document.querySelector('#node-management-modal h4');
        const addButton = this.manager.elements.addNodeBtn;

        if (formTitle) formTitle.textContent = `Edit Node: ${nodeName}`;
        if (addButton) {
            addButton.innerHTML = '<span class="material-symbols-outlined">save</span><span>Update Node</span>';
            addButton.onclick = () => this.updateNode();
        }

        this.manager.showNotification(`Edit mode for node ${nodeName}`, 'info');
    }

    async updateNode() {
        if (!this.editNodeName) {
            this.manager.showNotification('No node selected for editing', 'error');
            return;
        }

        const nodeData = {
            host: this.manager.elements.nodeHost.value,
            user: this.manager.elements.nodeUser.value,
            key_path: this.manager.elements.nodeKeypath.value,
            conf_dir: this.manager.elements.nodeConfdir.value,
            binary_dir: this.manager.elements.nodeBindir.value,
            description: this.manager.elements.nodeDescription.value,
            enabled: this.manager.elements.nodeEnabled.checked
        };

        if (!nodeData.host || !nodeData.user || !nodeData.key_path || !nodeData.conf_dir || !nodeData.binary_dir) {
            this.manager.showNotification('Please fill in all required fields', 'error');
            return;
        }

        this.setButtonLoading(this.manager.elements.addNodeBtn, true);

        try {
            // For now, we'll use PUT method to update the node
            // Note: This would need backend support for updating nodes
            const response = await this.manager.callAPI(`/api/nodes/${this.editNodeName}`, 'PUT', nodeData);

            if (response.success) {
                this.manager.showNotification(`Node ${this.editNodeName} updated successfully`, 'success');
                this.clearNodeForm();
                this.refreshNodesTable();
                this.manager.loadNodes(); // Refresh the dashboard nodes too
                this.exitEditMode();
            } else {
                throw new Error(response.message || 'Failed to update node');
            }
        } catch (error) {
            console.error('Error updating node:', error);
            this.manager.showNotification('Failed to update node: ' + error.message, 'error');
        } finally {
            this.setButtonLoading(this.manager.elements.addNodeBtn, false);
        }
    }

    exitEditMode() {
        this.isEditMode = false;
        this.editNodeName = null;

        // Reset form title and button
        const formTitle = document.querySelector('#node-management-modal h4');
        const addButton = this.manager.elements.addNodeBtn;

        if (formTitle) formTitle.textContent = 'Add New Node';
        if (addButton) {
            addButton.innerHTML = '<span class="material-symbols-outlined">add</span><span>Add Node</span>';
            addButton.onclick = () => this.addNode();
        }
    }

    setButtonLoading(button, loading) {
        if (!button) return;

        if (loading) {
            button.disabled = true;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Processing...</span>';
        } else {
            button.disabled = false;
            if (this.isEditMode) {
                button.innerHTML = '<span class="material-symbols-outlined">save</span><span>Update Node</span>';
            } else {
                button.innerHTML = '<span class="material-symbols-outlined">add</span><span>Add Node</span>';
            }
        }
    }
}
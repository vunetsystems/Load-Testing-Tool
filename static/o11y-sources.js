// O11y Sources Management Module
class O11ySources {
    constructor(manager) {
        this.manager = manager;
        this.o11ySources = [];
        this.selectedO11ySources = [];
        this.currentMode = 'custom'; // 'custom' or 'category'
        this.selectedCategory = null;

        // Define category mappings
        this.categories = {
            category1: {
                name: 'Category 1 (Azure Services)',
                sources: ['Azure_Firewall', 'Azure_Redis_Cache'],
                description: 'Azure Firewall and Azure Redis Cache monitoring'
            },
            category2: {
                name: 'Category 2 (Web & Database)',
                sources: ['Apache', 'MongoDB'],
                description: 'Apache web server and MongoDB database monitoring'
            },
            category3: {
                name: 'Category 3 (System & Database)',
                sources: ['LinuxMonitor', 'Mssql'],
                description: 'Linux system monitoring and Microsoft SQL Server'
            }
        };
    }

    loadO11ySources() {
        console.log('Loading o11y sources...');
        this.manager.callAPI('/api/o11y/sources')
            .then(response => {
                console.log('O11y sources API response:', response);
                if (response.success && response.data) {
                    console.log('Sources data:', response.data);
                    this.populateO11ySourcesSelect(response.data);
                    this.initializeModeSwitching();
                    console.log('Loaded o11y sources:', response.data.length, 'sources');
                } else {
                    console.error('Failed to load o11y sources:', response.message);
                    this.manager.showNotification('Failed to load O11y sources: ' + response.message, 'error');
                }
            })
            .catch(error => {
                console.error('Error loading o11y sources:', error);
                this.manager.showNotification('Error loading O11y sources: ' + error.message, 'error');
            });
    }

    populateO11ySourcesSelect(sources) {
        console.log('populateO11ySourcesSelect called with:', sources);
        const list = this.manager.elements.o11ySourcesList;
        if (!list) {
            console.error('o11ySourcesList element not found!');
            return;
        }

        console.log('o11ySourcesList element found:', list);

        // Store sources for filtering
        this.o11ySources = sources;

        // Clear existing options
        list.innerHTML = '';

        if (!sources || sources.length === 0) {
            console.error('No sources provided or sources array is empty');
            list.innerHTML = '<div class="o11y-sources-empty"><span class="material-symbols-outlined">error</span><p>No O11y sources available</p></div>';
            return;
        }

        // Add sources as custom options with checkboxes
        sources.forEach((source, index) => {
            console.log(`Adding source ${index + 1}: ${source}`);
            const option = document.createElement('div');
            option.className = 'o11y-source-option';
            option.innerHTML = `
                <div class="o11y-source-checkbox" data-source="${source}"></div>
                <span class="o11y-source-label">${source}</span>
            `;

            option.addEventListener('click', () => {
                console.log('Source clicked:', source);
                this.toggleO11ySource(source);
            });
            list.appendChild(option);
        });

        // Initialize selected sources array
        this.selectedO11ySources = [];

        console.log(`Successfully populated ${sources.length} o11y sources`);
    }

    syncConfigs() {
        const selectedSources = [...this.selectedO11ySources];
        const selectedEPS = parseInt(this.manager.elements.epsSelect.value);

        // Validate inputs before making API calls
        if (selectedSources.length === 0) {
            this.manager.showNotification('Please select at least one o11y source', 'warning');
            return;
        }

        if (!selectedEPS || selectedEPS <= 0) {
            this.manager.showNotification('Please select a valid EPS target greater than 0', 'warning');
            return;
        }

        if (selectedEPS > 1000000) {
            this.manager.showNotification('EPS target seems too high. Please verify the value.', 'warning');
            return;
        }

        // Show loading state
        this.setSyncButtonLoading(true);

        console.log('Syncing configs for sources:', selectedSources, 'EPS:', selectedEPS);

        // Call EPS distribution API first
       // Call EPS distribution API first
        this.manager.callAPI('/api/o11y/eps/distribute', 'POST', {
            selectedSources: selectedSources,
            totalEps: selectedEPS
        })
        .then(epsResponse => {
            console.log('EPS distribution response:', epsResponse);
            console.log('EPS distribution success:', epsResponse.success);

            if (!epsResponse.success) {
                // Provide specific error message for EPS distribution failure
                let epsErrorMessage = epsResponse.message || 'EPS distribution failed';
                console.error('EPS distribution failed with message:', epsErrorMessage);

                if (epsErrorMessage.includes('Total EPS must be greater than 0')) {
                    epsErrorMessage = 'Invalid EPS value. Please enter a value greater than 0.';
                } else if (epsErrorMessage.includes('no sources selected')) {
                    epsErrorMessage = 'No o11y sources selected. Please select at least one source.';
                } else if (epsErrorMessage.includes('max EPS not configured')) {
                    epsErrorMessage = 'EPS configuration not found for selected sources. Please check your configuration.';
                } else if (epsErrorMessage.includes('exceeds maximum limits')) {
                    epsErrorMessage = 'Selected EPS exceeds maximum allowed limits for one or more sources.';
                }

                // Immediately show error and stop processing - NO conf.d distribution will be called
                console.log('Stopping sync process due to EPS distribution failure');
                this.showSyncError(epsErrorMessage);
                this.manager.showNotification('Failed to sync configs: ' + epsErrorMessage, 'error');

                // Return a rejected promise to stop the chain
                return Promise.reject(new Error(epsErrorMessage));
            }

            // Only proceed to conf.d distribution if EPS distribution succeeded
            console.log('EPS distribution successful, proceeding to conf.d distribution...');
            console.log('About to call conf.d distribution API...');
            return this.manager.callAPI('/api/o11y/confd/distribute', 'POST');
        })
        .then(confDResponse => {
            console.log('Conf.d distribution response:', confDResponse);

            // Check for partial success in conf.d distribution
            if (confDResponse.data && confDResponse.data.failedNodes && confDResponse.data.failedNodes.length > 0) {
                const failedNodes = confDResponse.data.failedNodes;
                const totalNodes = confDResponse.data.totalNodes;
                const successCount = confDResponse.data.distributedNodes;

                if (successCount === 0) {
                    // Complete failure
                    throw new Error(`Configuration distribution failed completely. All ${totalNodes} nodes failed: ${failedNodes.join(', ')}`);
                } else {
                    // Partial success - show warning but don't fail completely
                    const warningMessage = `Configuration partially synced. ${successCount}/${totalNodes} nodes successful. Failed nodes: ${failedNodes.join(', ')}`;
                    this.showSyncSuccess();
                    this.manager.showNotification(warningMessage, 'warning');
                    console.warn('Partial sync success:', warningMessage);
                    return; // Exit early with partial success
                }
            }

            if (!confDResponse.success) {
                throw new Error(confDResponse.message || 'Conf.d distribution failed');
            }

            // Both APIs succeeded completely
            this.showSyncSuccess();
            this.manager.showNotification('Configs synced successfully!', 'success');
        })
        .catch(error => {
            console.error('Error syncing configs:', error);
            console.error('Error stack:', error.stack);

            // Ensure button is reset even if error occurs
            this.setSyncButtonLoading(false);

            // Provide more specific error messages based on error content
            let userFriendlyMessage = error.message;

            if (error.message.includes('EPS distribution failed')) {
                userFriendlyMessage = 'Failed to distribute EPS settings. Please check your o11y source configuration and EPS values.';
            } else if (error.message.includes('Conf.d distribution failed')) {
                userFriendlyMessage = 'Failed to distribute configuration files to nodes. Please check node connectivity.';
            } else if (error.message.includes('SSH connection failed')) {
                userFriendlyMessage = 'Unable to connect to nodes via SSH. Please check node credentials and network connectivity.';
            } else if (error.message.includes('all nodes failed')) {
                userFriendlyMessage = 'Configuration sync failed on all nodes. Please check node status and try again.';
            } else if (error.message.includes('HTTP error!')) {
                userFriendlyMessage = 'Network error occurred. Please check your connection and try again.';
            } else if (!userFriendlyMessage || userFriendlyMessage === 'undefined') {
                userFriendlyMessage = 'An unexpected error occurred during configuration sync. Please try again.';
            }

            this.showSyncError(userFriendlyMessage);
            this.manager.showNotification('Failed to sync configs: ' + userFriendlyMessage, 'error');
        })
        .finally(() => {
            console.log('Sync configs operation completed (success or failure)');
            this.setSyncButtonLoading(false);
        });
    }

    setSyncButtonLoading(loading) {
        const button = this.manager.elements.syncConfigsBtn;
        if (!button) return;

        if (loading) {
            button.disabled = true;
            button.innerHTML = '<span class="material-symbols-outlined animate-spin">sync</span><span>Syncing...</span>';
        } else {
            button.disabled = false;
            button.innerHTML = '<span class="material-symbols-outlined">sync</span><span>Sync Configs</span>';
        }
    }

    showSyncSuccess() {
        this.hideSyncMessages();
        this.manager.elements.syncSuccessMessage.classList.remove('hidden');
        this.manager.elements.syncStatusContainer.classList.remove('hidden');

        // Auto-hide after 5 seconds
        setTimeout(() => {
            this.hideSyncMessages();
        }, 5000);
    }

    showSyncError(message) {
        this.hideSyncMessages();
        this.manager.elements.syncErrorMessage.classList.remove('hidden');
        this.manager.elements.syncStatusContainer.classList.remove('hidden');

        // Auto-hide after 8 seconds for errors
        setTimeout(() => {
            this.hideSyncMessages();
        }, 8000);
    }

    hideSyncMessages() {
        this.manager.elements.syncSuccessMessage.classList.add('hidden');
        this.manager.elements.syncErrorMessage.classList.add('hidden');
        this.manager.elements.syncStatusContainer.classList.add('hidden');
    }

    // Custom Multi-Select Methods

    toggleO11ySourcesDropdown() {
        const isOpen = !this.manager.elements.o11ySourcesOptions.classList.contains('hidden');

        if (isOpen) {
            this.closeO11ySourcesDropdown();
        } else {
            this.openO11ySourcesDropdown();
        }
    }

    openO11ySourcesDropdown() {
        this.manager.elements.o11ySourcesOptions.classList.remove('hidden');
        this.manager.elements.o11ySourcesContainer.classList.add('open');
        this.manager.elements.o11ySourcesSearch.focus();

        // Update checkboxes to reflect current selection
        this.updateO11ySourceCheckboxes();
    }

    closeO11ySourcesDropdown() {
        this.manager.elements.o11ySourcesOptions.classList.add('hidden');
        this.manager.elements.o11ySourcesContainer.classList.remove('open');
        this.manager.elements.o11ySourcesSearch.value = '';
        this.filterO11ySources(''); // Show all sources
    }

    toggleO11ySource(source) {
        const index = this.selectedO11ySources.indexOf(source);

        if (index > -1) {
            this.selectedO11ySources.splice(index, 1);
        } else {
            this.selectedO11ySources.push(source);
        }

        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

    updateO11ySourceDisplay() {
        const selectedContainer = this.manager.elements.o11ySourcesSelected;
        const placeholder = this.manager.elements.o11ySourcesPlaceholder;

        // Clear current selection display
        selectedContainer.innerHTML = '';

        if (this.selectedO11ySources.length === 0) {
            placeholder.textContent = 'Select O11y sources...';
            selectedContainer.classList.add('hidden');
        } else {
            placeholder.textContent = `${this.selectedO11ySources.length} source${this.selectedO11ySources.length > 1 ? 's' : ''} selected`;
            selectedContainer.classList.remove('hidden');

            // Add selected items as removable tags
            this.selectedO11ySources.forEach(source => {
                const tag = document.createElement('div');
                tag.className = 'o11y-selected-item';
                tag.innerHTML = `
                    <span>${source}</span>
                    <span class="o11y-selected-item-remove material-symbols-outlined" data-source="${source}">close</span>
                `;

                tag.querySelector('.o11y-selected-item-remove').addEventListener('click', (e) => {
                    e.stopPropagation();
                    this.toggleO11ySource(source);
                });

                selectedContainer.appendChild(tag);
            });
        }
    }

    updateO11ySourceCheckboxes() {
        // Update all checkboxes to reflect current selection
        const checkboxes = this.manager.elements.o11ySourcesList.querySelectorAll('.o11y-source-checkbox');
        checkboxes.forEach(checkbox => {
            const source = checkbox.dataset.source;
            const isSelected = this.selectedO11ySources.includes(source);

            if (isSelected) {
                checkbox.classList.add('checked');
            } else {
                checkbox.classList.remove('checked');
            }
        });
    }

    updateO11ySourceCount() {
        const countElement = this.manager.elements.o11ySourcesCount;
        const totalSources = this.o11ySources.length;
        const selectedCount = this.selectedO11ySources.length;

        countElement.textContent = `${selectedCount}/${totalSources} selected`;
    }

    filterO11ySources(searchTerm) {
        const options = this.manager.elements.o11ySourcesList.querySelectorAll('.o11y-source-option');
        const term = searchTerm.toLowerCase();

        options.forEach(option => {
            const label = option.querySelector('.o11y-source-label');
            const source = label.textContent.toLowerCase();

            if (source.includes(term)) {
                option.style.display = 'flex';
                // Highlight search term
                if (term && source.includes(term)) {
                    const regex = new RegExp(`(${term})`, 'gi');
                    label.innerHTML = label.textContent.replace(regex, '<span class="search-highlight">$1</span>');
                } else {
                    label.innerHTML = label.textContent;
                }
            } else {
                option.style.display = 'none';
            }
        });

        // Show empty state if no results
        const visibleOptions = Array.from(options).filter(opt => opt.style.display !== 'none');
        const emptyState = this.manager.elements.o11ySourcesList.querySelector('.o11y-sources-empty');

        if (visibleOptions.length === 0) {
            if (!emptyState) {
                const empty = document.createElement('div');
                empty.className = 'o11y-sources-empty';
                empty.innerHTML = `
                    <span class="material-symbols-outlined">search_off</span>
                    <p>No sources found matching "${searchTerm}"</p>
                `;
                this.manager.elements.o11ySourcesList.appendChild(empty);
            }
        } else if (emptyState) {
            emptyState.remove();
        }
    }

    selectAllO11ySources() {
        this.selectedO11ySources = [...this.o11ySources];
        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

    clearAllO11ySources() {
        this.selectedO11ySources = [];
        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();
    }

    // Category-based selection methods

    initializeModeSwitching() {
        const modeSelect = this.manager.elements.o11ySourceMode;
        const categoryModeDiv = this.manager.elements.o11yCategoryMode;
        const categorySelect = this.manager.elements.o11yCategorySelect;

        if (!modeSelect || !categoryModeDiv || !categorySelect) {
            console.error('Mode switching elements not found');
            return;
        }

        // Set up mode change handler
        modeSelect.addEventListener('change', (e) => {
            this.switchMode(e.target.value);
        });

        // Set up category selection handler
        categorySelect.addEventListener('change', (e) => {
            this.selectCategory(e.target.value);
        });

        // Initialize to custom mode
        this.switchMode('custom');
    }

    switchMode(mode) {
        console.log('Switching mode to:', mode);
        this.currentMode = mode;

        const modeSelect = this.manager.elements.o11ySourceMode;
        const categoryModeDiv = this.manager.elements.o11yCategoryMode;
        const customModeDiv = this.manager.elements.o11ySourcesContainer;

        if (mode === 'category') {
            // Show category mode, hide custom mode
            categoryModeDiv.classList.remove('hidden');
            customModeDiv.classList.add('hidden');
            modeSelect.value = 'category';

            // Clear custom selections when switching to category mode
            this.clearAllO11ySources();
        } else {
            // Show custom mode, hide category mode
            categoryModeDiv.classList.add('hidden');
            customModeDiv.classList.remove('hidden');
            modeSelect.value = 'custom';

            // Clear category selection when switching to custom mode
            this.deselectCategory();
        }
    }

    initializeCategoryHandlers() {
        // Add click handlers to category options
        Object.keys(this.categories).forEach(categoryKey => {
            const categoryElement = document.getElementById(categoryKey);
            if (categoryElement) {
                categoryElement.addEventListener('click', () => {
                    this.selectCategory(categoryKey);
                });
            }
        });
    }

    selectCategory(categoryKey) {
        console.log('Selecting category:', categoryKey);

        if (!categoryKey || !this.categories[categoryKey]) {
            this.deselectCategory();
            return;
        }

        this.selectedCategory = categoryKey;
        const category = this.categories[categoryKey];

        // Update selected category info display
        this.updateSelectedCategoryInfo(category);

        // Select the corresponding o11y sources
        this.selectSourcesForCategory(category.sources);
    }

    updateSelectedCategoryInfo(category) {
        const infoElement = this.manager.elements.selectedCategoryInfo;
        const nameElement = this.manager.elements.selectedCategoryName;
        const sourcesElement = this.manager.elements.selectedCategorySources;

        if (infoElement && nameElement && sourcesElement) {
            nameElement.textContent = category.name;
            sourcesElement.textContent = category.sources.join(', ');
            infoElement.classList.remove('hidden');
        }
    }

    deselectCategory() {
        console.log('Deselecting category');
        this.selectedCategory = null;

        // Reset category dropdown
        const categorySelect = this.manager.elements.o11yCategorySelect;
        if (categorySelect) {
            categorySelect.value = '';
        }

        // Clear selections
        this.clearAllO11ySources();

        // Hide selected category info
        const infoElement = this.manager.elements.selectedCategoryInfo;
        if (infoElement) {
            infoElement.classList.add('hidden');
        }
    }

    selectSourcesForCategory(sourceNames) {
        console.log('Selecting sources for category:', sourceNames);

        // Clear existing selections
        this.selectedO11ySources = [];

        // Select sources that exist in our available sources
        sourceNames.forEach(sourceName => {
            if (this.o11ySources.includes(sourceName)) {
                this.selectedO11ySources.push(sourceName);
            } else {
                console.warn(`Source ${sourceName} not found in available sources`);
            }
        });

        // Update UI
        this.updateO11ySourceDisplay();
        this.updateO11ySourceCheckboxes();
        this.updateO11ySourceCount();

        console.log(`Selected ${this.selectedO11ySources.length} sources for category`);
    }

    getSelectedCategory() {
        return this.selectedCategory;
    }

    getCurrentMode() {
        return this.currentMode;
    }
}
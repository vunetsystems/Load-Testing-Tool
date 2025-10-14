package o11y_source_manager

import (
	"fmt"
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"vuDataSim/src/node_control"
	"gopkg.in/yaml.v3"
)

// O11ySourceManager manages observability source configurations and EPS distribution
type O11ySourceManager struct {
	configsDir   string
	maxEPSConfig MaxEPSConfig
	mainConfig   MainConfig
}

// MaxEPSConfig represents the maximum EPS configuration for each o11y source
type MaxEPSConfig struct {
	MaxEPS map[string]int `yaml:"max_eps_config"`
}

// MainConfig represents the main conf.d/conf.yml configuration
type MainConfig struct {
	IncludeModuleDirs map[string]ModuleDirConfig `yaml:"include_module_dirs"`
}

// SourceConfig represents an individual o11y source main configuration
type SourceConfig struct {
	Enabled           bool     `yaml:"enabled"`
	UniqueKey         UniqueKey `yaml:"uniquekey"`
	IncludeSubModules []string `yaml:"Include_sub_modules"`
}

type ModuleDirConfig struct {
    Enabled bool `yaml:"enabled"`
}

// UniqueKey represents the uniquekey configuration
type UniqueKey struct {
	Name       string `yaml:"name"`
	DataType   string `yaml:"DataType"`
	ValueType  string `yaml:"ValueType"`
	Value      string `yaml:"Value"`
	NumUniqKey int    `yaml:"NumUniqKey"`
}

// SubModuleConfig represents a submodule configuration
type SubModuleConfig struct {
	UniqueKey UniqueKey `yaml:"uniquekey"`
}

// EPSDistributionRequest represents a request to distribute EPS across o11y sources
type EPSDistributionRequest struct {
	SelectedSources []string `json:"selectedSources"`
	TotalEPS        int      `json:"totalEps"`
}

// EPSDistributionResponse represents the response after EPS distribution
type EPSDistributionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// SourceEPSInfo represents EPS information for a source
type SourceEPSInfo struct {
	SourceName     string         `json:"sourceName"`
	AssignedEPS    int            `json:"assignedEps"`
	MainUniqueKeys int            `json:"mainUniqueKeys"`
	TotalSubKeys   int            `json:"totalSubKeys"`
	SubModuleKeys  map[string]int `json:"subModuleKeys"`
}

// NewO11ySourceManager creates a new O11ySourceManager instance
func NewO11ySourceManager() *O11ySourceManager {
	return &O11ySourceManager{
		configsDir:   "src/configs",
		maxEPSConfig: MaxEPSConfig{MaxEPS: make(map[string]int)},
		mainConfig:   MainConfig{},
	}
}

// LoadMaxEPSConfig loads the maximum EPS configuration from YAML file
func (osm *O11ySourceManager) LoadMaxEPSConfig() error {
	configPath := filepath.Join(osm.configsDir, "max_eps.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("max EPS config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read max EPS config file: %v", err)
	}

	err = yaml.Unmarshal(data, &osm.maxEPSConfig)
	if err != nil {
		return fmt.Errorf("failed to parse max EPS config file: %v", err)
	}

	log.Printf("Loaded max EPS config for %d sources", len(osm.maxEPSConfig.MaxEPS))
	return nil
}

// LoadMainConfig loads the main configuration from conf.d/conf.yml
func (osm *O11ySourceManager) LoadMainConfig() error {
	configPath := "src/conf.d/conf.yml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("main config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read main config file: %v", err)
	}

	err = yaml.Unmarshal(data, &osm.mainConfig)
	if err != nil {
		return fmt.Errorf("failed to parse main config file: %v", err)
	}

	log.Println("Loaded main configuration")
	return nil
}

// GetAvailableSources returns a list of all available o11y sources
func (osm *O11ySourceManager) GetAvailableSources() []string {
	var sources []string
	for sourceName := range osm.maxEPSConfig.MaxEPS {
		sources = append(sources, sourceName)
	}
	sort.Strings(sources)
	return sources
}

// GetEnabledSources returns a list of currently enabled o11y sources
func (osm *O11ySourceManager) GetEnabledSources() []string {
	var sources []string
	for sourceName, config := range osm.mainConfig.IncludeModuleDirs {
		if config.Enabled {
			sources = append(sources, sourceName)
		}
	}
	sort.Strings(sources)
	return sources
}

// DistributeEPS distributes the total EPS across selected sources proportionally
func (osm *O11ySourceManager) DistributeEPS(request EPSDistributionRequest) (*EPSDistributionResponse, error) {
	// Validate request
	if request.TotalEPS <= 0 {
		return &EPSDistributionResponse{
			Success: false,
			Message: "Total EPS must be greater than 0",
		}, fmt.Errorf("invalid total EPS: %d", request.TotalEPS)
	}

	if len(request.SelectedSources) == 0 {
		return &EPSDistributionResponse{
			Success: false,
			Message: "At least one source must be selected",
		}, fmt.Errorf("no sources selected")
	}

	// Calculate proportional distribution
	sourceEPSMap, err := osm.calculateProportionalDistribution(request.SelectedSources, request.TotalEPS)
	if err != nil {
		return &EPSDistributionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to calculate distribution: %v", err),
		}, err
	}

	// Apply the distribution
	err = osm.applyEPSDistribution(sourceEPSMap)
	if err != nil {
		return &EPSDistributionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to apply distribution: %v", err),
		}, err
	}

	// Prepare response data with new NumUniqKey values
	responseData := map[string]interface{}{
		"totalEps":        request.TotalEPS,
		"selectedSources": request.SelectedSources,
		"sourceBreakdown": osm.getSourceEPSBreakdown(),
		"newTotalEps":     osm.calculateCurrentEPS(),
		"updatedConfigs":  osm.getUpdatedNumUniqKeyValues(request.SelectedSources),
	}

	return &EPSDistributionResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully distributed %d EPS across %d sources", request.TotalEPS, len(request.SelectedSources)),
		Data:    responseData,
	}, nil
}

// calculateProportionalDistribution calculates EPS distribution based on max EPS values
func (osm *O11ySourceManager) calculateProportionalDistribution(selectedSources []string, totalEPS int) (map[string]int, error) {
	if len(selectedSources) == 0 {
		return nil, fmt.Errorf("no sources selected")
	}

	// Calculate total max EPS for selected sources
	totalMaxEPS := 0
	sourceMaxEPS := make(map[string]int)

	for _, sourceName := range selectedSources {
		maxEPS, exists := osm.maxEPSConfig.MaxEPS[sourceName]
		if !exists {
			return nil, fmt.Errorf("max EPS not configured for source: %s", sourceName)
		}
		sourceMaxEPS[sourceName] = maxEPS
		totalMaxEPS += maxEPS
	}

	if totalMaxEPS == 0 {
		return nil, fmt.Errorf("total max EPS is 0 for selected sources")
	}

	// Distribute EPS proportionally
	sourceEPSMap := make(map[string]int)
	remainingEPS := totalEPS

	for i, sourceName := range selectedSources {
		if i == len(selectedSources)-1 {
			// Last source gets remaining EPS to avoid rounding issues
			sourceEPSMap[sourceName] = remainingEPS
		} else {
			// Calculate proportional EPS
			proportion := float64(sourceMaxEPS[sourceName]) / float64(totalMaxEPS)
			assignedEPS := int(float64(totalEPS) * proportion)
			sourceEPSMap[sourceName] = assignedEPS
			remainingEPS -= assignedEPS
		}
	}

	return sourceEPSMap, nil
}

// applyEPSDistribution applies the calculated EPS distribution to source configurations
// applyEPSDistribution applies the calculated EPS distribution to source configurations
func (osm *O11ySourceManager) applyEPSDistribution(sourceEPSMap map[string]int) error {
	log.Printf("DEBUG: Starting applyEPSDistribution with %d sources", len(sourceEPSMap))
	log.Printf("DEBUG: Current IncludeModuleDirs before processing has %d entries", len(osm.mainConfig.IncludeModuleDirs))
	
	// Ensure the map is initialized
	if osm.mainConfig.IncludeModuleDirs == nil {
		log.Println("DEBUG: IncludeModuleDirs is nil, initializing...")
		osm.mainConfig.IncludeModuleDirs = make(map[string]ModuleDirConfig)
	}
	
	// Get all available sources from max EPS config to ensure we have all sources in the map
	for sourceName := range osm.maxEPSConfig.MaxEPS {
		if _, exists := osm.mainConfig.IncludeModuleDirs[sourceName]; !exists {
			log.Printf("DEBUG: Adding missing source %s to IncludeModuleDirs", sourceName)
			osm.mainConfig.IncludeModuleDirs[sourceName] = ModuleDirConfig{Enabled: false}
		}
	}
	
	log.Printf("DEBUG: After ensuring all sources exist, IncludeModuleDirs has %d entries", len(osm.mainConfig.IncludeModuleDirs))
	
	// First, disable ALL sources in main config
	for sourceName := range osm.mainConfig.IncludeModuleDirs {
		config := osm.mainConfig.IncludeModuleDirs[sourceName]
		config.Enabled = false
		osm.mainConfig.IncludeModuleDirs[sourceName] = config
		log.Printf("DEBUG: Disabled source: %s", sourceName)
	}

	// Then, enable ONLY the selected sources
	for sourceName := range sourceEPSMap {
		// Calculate total submodule keys for this source
		totalSubKeys := osm.calculateTotalSubModuleKeys(sourceName)
		if totalSubKeys == 0 {
			totalSubKeys = 1 // Avoid division by zero
		}

		// Calculate required main unique keys
		assignedEPS := sourceEPSMap[sourceName]
		requiredMainKeys := assignedEPS / totalSubKeys
		if requiredMainKeys <= 0 {
			requiredMainKeys = 1
		}

		// Update the source configuration
		err := osm.updateSourceConfig(sourceName, requiredMainKeys)
		if err != nil {
			return fmt.Errorf("failed to update config for source %s: %v", sourceName, err)
		}
		
		// Enable this source in main config
		config := osm.mainConfig.IncludeModuleDirs[sourceName]
		config.Enabled = true
		osm.mainConfig.IncludeModuleDirs[sourceName] = config
		log.Printf("DEBUG: Enabled source: %s", sourceName)

		log.Printf("Updated %s: EPS=%d, MainKeys=%d, SubKeys=%d, Enabled=true",
			sourceName, assignedEPS, requiredMainKeys, totalSubKeys)
	}
	
	log.Printf("DEBUG: After enabling selected sources, IncludeModuleDirs has %d entries", len(osm.mainConfig.IncludeModuleDirs))
	log.Printf("DEBUG: About to call saveMainConfig...")

	// Save the updated main configuration
	return osm.saveMainConfig()
}

// calculateTotalSubModuleKeys calculates total submodule unique keys for a source
func (osm *O11ySourceManager) calculateTotalSubModuleKeys(sourceName string) int {
	totalKeys := 0
	sourcePath := filepath.Join("src/conf.d", sourceName)

	// Load source config to get submodule list
	configPath := filepath.Join(sourcePath, "conf.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Warning: Source config not found: %s", configPath)
		return 1
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Warning: Failed to read source config %s: %v", configPath, err)
		return 1
	}

	var sourceConfig SourceConfig
	err = yaml.Unmarshal(data, &sourceConfig)
	if err != nil {
		log.Printf("Warning: Failed to parse source config %s: %v", configPath, err)
		return 1
	}

	// Process each submodule
	for _, subModuleName := range sourceConfig.IncludeSubModules {
		// Handle array format (with or without brackets)
		subModuleName = strings.TrimSpace(strings.Trim(subModuleName, "[]"))
		if subModuleName == "" {
			continue
		}

		subModulePath := filepath.Join(sourcePath, subModuleName+".yml")
		if _, err := os.Stat(subModulePath); os.IsNotExist(err) {
			log.Printf("Warning: Submodule file not found: %s", subModulePath)
			totalKeys += 1 // Count as 1 if file doesn't exist
			continue
		}

		data, err := os.ReadFile(subModulePath)
		if err != nil {
			log.Printf("Warning: Failed to read submodule file %s: %v", subModulePath, err)
			totalKeys += 1 // Count as 1 if read fails
			continue
		}

		var subModuleConfig SubModuleConfig
		err = yaml.Unmarshal(data, &subModuleConfig)
		if err != nil {
			log.Printf("Warning: Failed to parse submodule file %s: %v", subModulePath, err)
			totalKeys += 1 // Count as 1 if parse fails
			continue
		}

		// Add to total (use 1 if uniquekey doesn't exist or NumUniqKey is 0)
		if subModuleConfig.UniqueKey.NumUniqKey > 0 {
			totalKeys += subModuleConfig.UniqueKey.NumUniqKey
		} else {
			totalKeys += 1
		}
	}

	return totalKeys
}

// updateSourceConfig updates the NumUniqKey field in a source's conf.yml file
func (osm *O11ySourceManager) updateSourceConfig(sourceName string, numUniqKey int) error {
	configPath := filepath.Join("src/conf.d", sourceName, "conf.yml")

	// Read file as text to preserve formatting
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	text := string(data)

	// Simple string replacement - find and replace NumUniqKey value
	if strings.Contains(text, "NumUniqKey:") {
		lines := strings.Split(text, "\n")
		for i, line := range lines {
			if strings.Contains(line, "NumUniqKey:") && strings.Contains(line, "uniquekey:") == false {
				// This is the NumUniqKey line, replace just the number
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					lines[i] = parts[0] + ": " + fmt.Sprintf("%d", numUniqKey)
				}
			}
		}
		text = strings.Join(lines, "\n")
	}

	err = os.WriteFile(configPath, []byte(text), 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// saveMainConfig saves the main configuration to its YAML file.
// NOTE: This approach is more robust but will remove comments and reformat the file.
func (osm *O11ySourceManager) saveMainConfig() error {
    configPath := "src/conf.d/conf.yml"
    log.Printf("DEBUG: Attempting to save main config to %s", configPath)
    log.Printf("DEBUG: Current IncludeModuleDirs has %d entries", len(osm.mainConfig.IncludeModuleDirs))

    // --- Create a temporary structure to match the file's layout ---
    // This ensures the output YAML has the correct top-level keys.
    fullConfig := make(map[string]interface{})

    // Read the original file to get all top-level keys (like logging, output.kafka, etc.)
    data, err := os.ReadFile(configPath)
    if err != nil {
        log.Printf("DEBUG: Failed to read original config file to preserve keys: %v", err)
        return fmt.Errorf("failed to read main config file: %v", err)
    }
    
    // Unmarshal into a generic map to preserve all other sections
    err = yaml.Unmarshal(data, &fullConfig)
    if err != nil {
        log.Printf("DEBUG: Failed to unmarshal original config: %v", err)
        return fmt.Errorf("failed to unmarshal original main config: %v", err)
    }

    // --- Convert IncludeModuleDirs to the format expected by YAML marshaling ---
    // The issue is that the struct values need to be converted to map[string]interface{}
    moduleDirsMap := make(map[string]interface{})
    for sourceName, config := range osm.mainConfig.IncludeModuleDirs {
        moduleDirsMap[sourceName] = map[string]interface{}{
            "enabled": config.Enabled,
        }
    }
    
    log.Printf("DEBUG: Converted %d sources to map format", len(moduleDirsMap))

    // --- Overwrite the 'include_module_dirs' section with our updated data ---
    fullConfig["include_module_dirs"] = moduleDirsMap

    // --- Marshal the updated configuration map to YAML ---
    var buf bytes.Buffer
    encoder := yaml.NewEncoder(&buf)
    encoder.SetIndent(2) // Keep the indentation clean
    err = encoder.Encode(fullConfig)
    if err != nil {
        log.Printf("DEBUG: Failed to marshal updated config map: %v", err)
        return fmt.Errorf("failed to marshal updated main config: %v", err)
    }

    // --- Write the new YAML content to the file ---
    log.Println("DEBUG: YAML marshalled successfully. Writing back to file...")
    err = os.WriteFile(configPath, buf.Bytes(), 0644)
    if err != nil {
        log.Printf("DEBUG: FAILED to write updated main config file: %v", err)
        return fmt.Errorf("failed to write updated main config file: %v", err)
    }

    log.Println("DEBUG: Successfully saved main config file.")
    return nil
}

// calculateCurrentEPS calculates the current total EPS across all enabled sources
func (osm *O11ySourceManager) calculateCurrentEPS() int {
	totalEPS := 0
	for sourceName, config := range osm.mainConfig.IncludeModuleDirs {
		if config.Enabled {
			totalSubKeys := osm.calculateTotalSubModuleKeys(sourceName)
			if totalSubKeys == 0 {
				totalSubKeys = 1
			}

			// Load source config to get current NumUniqKey
			sourceConfig, err := osm.loadSourceConfig(sourceName)
			if err != nil {
				log.Printf("Warning: Failed to load source config for %s: %v", sourceName, err)
				continue
			}

			sourceEPS := sourceConfig.UniqueKey.NumUniqKey * totalSubKeys
			totalEPS += sourceEPS
		}
	}
	return totalEPS
}

// loadSourceConfig loads configuration for a specific o11y source
func (osm *O11ySourceManager) loadSourceConfig(sourceName string) (*SourceConfig, error) {
	configPath := filepath.Join("src/conf.d", sourceName, "conf.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("source config file not found: %s", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source config file: %v", err)
	}

	var config SourceConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source config file: %v", err)
	}

	return &config, nil
}

// getSourceEPSBreakdown returns detailed EPS breakdown for all sources
func (osm *O11ySourceManager) getSourceEPSBreakdown() map[string]SourceEPSInfo {
	breakdown := make(map[string]SourceEPSInfo)

	for sourceName, config := range osm.mainConfig.IncludeModuleDirs {
		if config.Enabled {
			totalSubKeys := osm.calculateTotalSubModuleKeys(sourceName)
			if totalSubKeys == 0 {
				totalSubKeys = 1
			}

			// Load source config to get current NumUniqKey
			sourceConfig, err := osm.loadSourceConfig(sourceName)
			if err != nil {
				log.Printf("Warning: Failed to load source config for %s: %v", sourceName, err)
				continue
			}

			eps := sourceConfig.UniqueKey.NumUniqKey * totalSubKeys

			info := SourceEPSInfo{
				SourceName:     sourceName,
				AssignedEPS:    eps,
				MainUniqueKeys: sourceConfig.UniqueKey.NumUniqKey,
				TotalSubKeys:   totalSubKeys,
				SubModuleKeys:  make(map[string]int),
			}

			// Add submodule breakdown
			for _, subModuleName := range sourceConfig.IncludeSubModules {
				subModuleName = strings.TrimSpace(strings.Trim(subModuleName, "[]"))
				if subModuleName == "" {
					continue
				}

				subModulePath := filepath.Join("src/conf.d", sourceName, subModuleName+".yml")
				if _, err := os.Stat(subModulePath); os.IsNotExist(err) {
					info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
					continue
				}

				data, err := os.ReadFile(subModulePath)
				if err != nil {
					info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
					continue
				}

				var subModuleConfig SubModuleConfig
				err = yaml.Unmarshal(data, &subModuleConfig)
				if err != nil {
					info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
					continue
				}

				subEPS := sourceConfig.UniqueKey.NumUniqKey
				if subModuleConfig.UniqueKey.NumUniqKey > 0 {
					subEPS *= subModuleConfig.UniqueKey.NumUniqKey
				}
				info.SubModuleKeys[subModuleName] = subEPS
			}

			breakdown[sourceName] = info
		}
	}

	return breakdown
}

// GetSourceDetails returns detailed information about a specific source
func (osm *O11ySourceManager) GetSourceDetails(sourceName string) (*SourceEPSInfo, error) {
	if _, exists := osm.maxEPSConfig.MaxEPS[sourceName]; !exists {
		return nil, fmt.Errorf("source not found: %s", sourceName)
	}

	totalSubKeys := osm.calculateTotalSubModuleKeys(sourceName)
	if totalSubKeys == 0 {
		totalSubKeys = 1
	}

	sourceConfig, err := osm.loadSourceConfig(sourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to load source config: %v", err)
	}

	eps := sourceConfig.UniqueKey.NumUniqKey * totalSubKeys

	info := SourceEPSInfo{
		SourceName:     sourceName,
		AssignedEPS:    eps,
		MainUniqueKeys: sourceConfig.UniqueKey.NumUniqKey,
		TotalSubKeys:   totalSubKeys,
		SubModuleKeys:  make(map[string]int),
	}

	// Add submodule breakdown
	for _, subModuleName := range sourceConfig.IncludeSubModules {
		subModuleName = strings.TrimSpace(strings.Trim(subModuleName, "[]"))
		if subModuleName == "" {
			continue
		}

		subModulePath := filepath.Join("src/conf.d", sourceName, subModuleName+".yml")
		if _, err := os.Stat(subModulePath); os.IsNotExist(err) {
			info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
			continue
		}

		data, err := os.ReadFile(subModulePath)
		if err != nil {
			info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
			continue
		}

		var subModuleConfig SubModuleConfig
		err = yaml.Unmarshal(data, &subModuleConfig)
		if err != nil {
			info.SubModuleKeys[subModuleName] = sourceConfig.UniqueKey.NumUniqKey
			continue
		}

		subEPS := sourceConfig.UniqueKey.NumUniqKey
		if subModuleConfig.UniqueKey.NumUniqKey > 0 {
			subEPS *= subModuleConfig.UniqueKey.NumUniqKey
		}
		info.SubModuleKeys[subModuleName] = subEPS
	}

	return &info, nil
}

// EnableSource enables a specific o11y source
func (osm *O11ySourceManager) EnableSource(sourceName string) error {
	if _, exists := osm.maxEPSConfig.MaxEPS[sourceName]; !exists {
		return fmt.Errorf("source not found: %s", sourceName)
	}

	if mainConfigEntry, exists := osm.mainConfig.IncludeModuleDirs[sourceName]; exists {
		mainConfigEntry.Enabled = true
		osm.mainConfig.IncludeModuleDirs[sourceName] = mainConfigEntry
	}

	return osm.saveMainConfig()
}

// DisableSource disables a specific o11y source
func (osm *O11ySourceManager) DisableSource(sourceName string) error {
	if _, exists := osm.maxEPSConfig.MaxEPS[sourceName]; !exists {
		return fmt.Errorf("source not found: %s", sourceName)
	}

	if mainConfigEntry, exists := osm.mainConfig.IncludeModuleDirs[sourceName]; exists {
		mainConfigEntry.Enabled = false
		osm.mainConfig.IncludeModuleDirs[sourceName] = mainConfigEntry
	}

	return osm.saveMainConfig()
}

// GetMaxEPSConfig returns the maximum EPS configuration
func (osm *O11ySourceManager) GetMaxEPSConfig() map[string]int {
	return osm.maxEPSConfig.MaxEPS
}

// GetSourceEPSBreakdown returns detailed EPS breakdown for all sources (public method)
func (osm *O11ySourceManager) GetSourceEPSBreakdown() map[string]SourceEPSInfo {
	return osm.getSourceEPSBreakdown()
}

// CalculateCurrentEPS calculates the current total EPS across all enabled sources (public method)
func (osm *O11ySourceManager) CalculateCurrentEPS() int {
	return osm.calculateCurrentEPS()
}

// getUpdatedNumUniqKeyValues returns the new NumUniqKey values for selected sources
func (osm *O11ySourceManager) getUpdatedNumUniqKeyValues(selectedSources []string) map[string]int {
	updatedValues := make(map[string]int)

	for _, sourceName := range selectedSources {
		// Load the source config to get the updated NumUniqKey value
		sourceConfig, err := osm.loadSourceConfig(sourceName)
		if err != nil {
			log.Printf("Warning: Failed to load source config for %s: %v", sourceName, err)
			continue
		}

		updatedValues[sourceName] = sourceConfig.UniqueKey.NumUniqKey
	}

	return updatedValues
}

// ConfDNodeResult represents the result of conf.d distribution to a single node
type ConfDNodeResult struct {
	NodeName string `json:"nodeName"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
}

// ConfDDistributionResponse represents the response after conf.d distribution
type ConfDDistributionResponse struct {
	Success           bool                        `json:"success"`
	Message           string                      `json:"message"`
	Data              map[string]interface{}      `json:"data,omitempty"`
	Distribution      map[string]ConfDNodeResult  `json:"distribution"`
}

// DistributeConfD distributes the conf.d directory to all enabled nodes
// DistributeConfD distributes the conf.d directory to all enabled nodes
// DistributeConfD distributes the conf.d directory to all enabled nodes
func (osm *O11ySourceManager) DistributeConfD() (*ConfDDistributionResponse, error) {
	log.Println("Starting conf.d distribution to all enabled nodes...")
	
	// Load node manager to access node configurations
	nodeManager := osm.getNodeManager()
	if nodeManager == nil {
		return &ConfDDistributionResponse{
			Success: false,
			Message: "Node manager not available",
		}, fmt.Errorf("node manager not available")
	}
	
	// Get enabled nodes
	enabledNodes := nodeManager.GetEnabledNodes()
	if len(enabledNodes) == 0 {
		log.Println("No enabled nodes found to distribute conf.d to")
		return &ConfDDistributionResponse{
			Success: true,
			Message: "No enabled nodes found to distribute conf.d to",
			Data: map[string]interface{}{
				"distributedNodes": 0,
				"totalNodes":       0,
				"successRate":     "0/0",
			},
			Distribution: make(map[string]ConfDNodeResult),
		}, nil
	}
	
	log.Printf("Found %d enabled nodes to distribute conf.d to", len(enabledNodes))
	
	// Create temporary tar file from local conf.d directory
	tempTarFile := "/tmp/confd_backup.tar.gz"
	localConfDir := "src/conf.d"
	
	// Check if local conf.d directory exists
	if _, err := os.Stat(localConfDir); os.IsNotExist(err) {
		return &ConfDDistributionResponse{
			Success: false,
			Message: fmt.Sprintf("Local conf.d directory not found: %s", localConfDir),
		}, fmt.Errorf("local conf.d directory not found: %s", localConfDir)
	}
	
	// Create tar command - include the conf.d directory itself
	tarCmd := exec.Command("tar", "-czf", tempTarFile, "-C", filepath.Dir(localConfDir), filepath.Base(localConfDir))
	log.Printf("Creating temporary tar file: tar -czf %s -C %s %s", tempTarFile, filepath.Dir(localConfDir), filepath.Base(localConfDir))
	
	if err := tarCmd.Run(); err != nil {
		return &ConfDDistributionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create tar file: %v", err),
		}, fmt.Errorf("failed to create tar file: %v", err)
	}
	
	defer func() {
		// Clean up temporary tar file
		if err := os.Remove(tempTarFile); err != nil {
			log.Printf("Warning: Failed to remove temporary tar file %s: %v", tempTarFile, err)
		}
	}()
	
	// Distribute to each enabled node
	distributionResults := make(map[string]ConfDNodeResult)
	successCount := 0
	
	for nodeName, nodeConfig := range enabledNodes {
		log.Printf("Distributing conf.d to node: %s (host: %s, conf_dir: %s)", nodeName, nodeConfig.Host, nodeConfig.ConfDir)
		
		result := osm.distributeConfDToNode(nodeName, nodeConfig, tempTarFile)
		distributionResults[nodeName] = result
		
		if result.Success {
			successCount++
			log.Printf("✓ Successfully distributed conf.d to node: %s", nodeName)
		} else {
			log.Printf("✗ Failed to distribute conf.d to node: %s - %s", nodeName, result.Message)
		}
	}
	
	successRate := fmt.Sprintf("%d/%d", successCount, len(enabledNodes))
	message := fmt.Sprintf("Conf.d distribution completed: %s nodes successful", successRate)
	
	response := &ConfDDistributionResponse{
		Success: successCount == len(enabledNodes),
		Message: message,
		Data: map[string]interface{}{
			"distributedNodes": successCount,
			"totalNodes":       len(enabledNodes),
			"successRate":      successRate,
		},
		Distribution: distributionResults,
	}
	
	log.Printf("✓ Conf.d distribution completed successfully to %d/%d nodes", successCount, len(enabledNodes))
	return response, nil
}

// distributeConfDToNode distributes conf.d to a single node
func (osm *O11ySourceManager) distributeConfDToNode(nodeName string, nodeConfig node_control.NodeConfig, tempTarFile string) ConfDNodeResult {
	log.Printf("Starting conf.d replacement for node %s", nodeConfig.Host)
	
	// nodeConfig.ConfDir is the parent directory where conf.d should be placed (e.g., /path/to/)
	// We need to create /path/to/conf.d
	targetConfDir := filepath.Join(nodeConfig.ConfDir, "conf.d")
	
	// Remove existing conf.d directory on remote node
	log.Printf("Removing existing conf.d directory on remote node: rm -rf %s", targetConfDir)
	err := osm.sshExec(nodeConfig, fmt.Sprintf("rm -rf %s", targetConfDir))
	if err != nil {
		return ConfDNodeResult{
			NodeName: nodeName,
			Success:  false,
			Message:  fmt.Sprintf("Failed to remove existing conf.d directory: %v", err),
		}
	}
	
	// Ensure parent directory exists
	log.Printf("Creating parent directory if needed: mkdir -p %s", nodeConfig.ConfDir)
	err = osm.sshExec(nodeConfig, fmt.Sprintf("mkdir -p %s", nodeConfig.ConfDir))
	if err != nil {
		return ConfDNodeResult{
			NodeName: nodeName,
			Success:  false,
			Message:  fmt.Sprintf("Failed to create parent directory: %v", err),
		}
	}
	
	// Copy tar file to a temporary location
	remoteTarPath := filepath.Join("/tmp", "confd_backup_"+nodeName+".tar.gz")
	log.Printf("Copying tar file to remote node: scp %s to %s", tempTarFile, remoteTarPath)
	err = osm.scpCopy(nodeConfig, tempTarFile, remoteTarPath)
	if err != nil {
		return ConfDNodeResult{
			NodeName: nodeName,
			Success:  false,
			Message:  fmt.Sprintf("Failed to copy tar file: %v", err),
		}
	}
	
	// Extract tar file to the target directory
	// The tar contains "conf.d/" so it will create conf.d in nodeConfig.ConfDir
	extractAndCleanupCmd := fmt.Sprintf(
		"cd %s && tar -xzf %s && rm %s",
		nodeConfig.ConfDir,
		remoteTarPath,
		remoteTarPath,
	)
	
	log.Printf("Extracting tar file on remote node: %s", extractAndCleanupCmd)
	err = osm.sshExec(nodeConfig, extractAndCleanupCmd)
	if err != nil {
		return ConfDNodeResult{
			NodeName: nodeName,
			Success:  false,
			Message:  fmt.Sprintf("Failed to extract tar file: %v", err),
		}
	}
	
	// Verify the conf.d directory exists in the target location
	verifyCmd := fmt.Sprintf("test -d %s", targetConfDir)
	log.Printf("Verifying conf.d directory exists at: %s", targetConfDir)
	err = osm.sshExec(nodeConfig, verifyCmd)
	if err != nil {
		// Additional debug: list the parent directory to see what was created
		listCmd := fmt.Sprintf("ls -la %s", nodeConfig.ConfDir)
		osm.sshExec(nodeConfig, listCmd)
		
		return ConfDNodeResult{
			NodeName: nodeName,
			Success:  false,
			Message:  fmt.Sprintf("Conf.d directory not found after extraction at %s: %v", targetConfDir, err),
		}
	}
	
	log.Printf("✓ Conf.d replacement completed for node %s at %s", nodeConfig.Host, targetConfDir)
	return ConfDNodeResult{
		NodeName: nodeName,
		Success:  true,
		Message:  fmt.Sprintf("Conf.d distributed successfully to %s", targetConfDir),
	}
}

// sshExec executes a command on the remote node via SSH
func (osm *O11ySourceManager) sshExec(nodeConfig node_control.NodeConfig, command string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		fmt.Sprintf("%s@%s", nodeConfig.User, nodeConfig.Host),
		command,
	}

	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SSH command failed: %v", err)
	}

	return nil
}

// scpCopy copies a file to the remote node
func (osm *O11ySourceManager) scpCopy(nodeConfig node_control.NodeConfig, localPath, remotePath string) error {
	args := []string{
		"-i", nodeConfig.KeyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=10",
		localPath,
		fmt.Sprintf("%s@%s:%s", nodeConfig.User, nodeConfig.Host, remotePath),
	}

	cmd := exec.Command("scp", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("SCP copy failed: %v", err)
	}

	return nil
}

// getNodeManager returns the node manager instance
// Note: This is a workaround since we can't directly access the global nodeManager from main.go
// In a production system, you might want to pass the nodeManager as a dependency
func (osm *O11ySourceManager) getNodeManager() *node_control.NodeManager {
	// For now, create a new instance - this is not ideal but works for the API
	// In a better design, the nodeManager would be injected as a dependency
	nodeManager := node_control.NewNodeManager()
	err := nodeManager.LoadNodesConfig()
	if err != nil {
		log.Printf("Warning: Failed to load nodes config in getNodeManager: %v", err)
		return nil
	}
	return nodeManager
}
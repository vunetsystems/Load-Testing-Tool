package o11y_source_manager

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	IncludeModuleDirs map[string]struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"include_module_dirs"`
}

// SourceConfig represents an individual o11y source main configuration
type SourceConfig struct {
	Enabled           bool     `yaml:"enabled"`
	UniqueKey         UniqueKey `yaml:"uniquekey"`
	IncludeSubModules []string `yaml:"Include_sub_modules"`
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
func (osm *O11ySourceManager) applyEPSDistribution(sourceEPSMap map[string]int) error {
	// First, disable ALL sources in main config
	for sourceName := range osm.mainConfig.IncludeModuleDirs {
		if mainConfigEntry, exists := osm.mainConfig.IncludeModuleDirs[sourceName]; exists {
			mainConfigEntry.Enabled = false
			osm.mainConfig.IncludeModuleDirs[sourceName] = mainConfigEntry
		}
	}

	// Then, enable ONLY the selected sources
	for sourceName := range sourceEPSMap {
		if mainConfigEntry, exists := osm.mainConfig.IncludeModuleDirs[sourceName]; exists {
			mainConfigEntry.Enabled = true
			osm.mainConfig.IncludeModuleDirs[sourceName] = mainConfigEntry
		}

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

		log.Printf("Updated %s: EPS=%d, MainKeys=%d, SubKeys=%d, Enabled=true",
			sourceName, assignedEPS, requiredMainKeys, totalSubKeys)
	}

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

// saveMainConfig saves the main configuration to its YAML file
func (osm *O11ySourceManager) saveMainConfig() error {
	configPath := "src/conf.d/conf.yml"

	// Read the existing file to preserve all other content
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read main config file: %v", err)
	}

	text := string(data)

	// Update only the enabled fields within include_module_dirs
	for sourceName, config := range osm.mainConfig.IncludeModuleDirs {
		newText := "    enabled: true"
		if !config.Enabled {
			newText = "    enabled: false"
		}

		// Find and replace the specific enabled line for this source
		// Pattern: sourceName:\n    enabled: (true|false)
		sourcePattern := sourceName + ":\n    enabled:"
		if strings.Contains(text, sourcePattern) {
			// Replace enabled: false with new value
			oldText := sourceName + ":\n    enabled: false"
			newFullText := sourceName + ":\n" + newText
			text = strings.Replace(text, oldText, newFullText, 1)

			// Also handle enabled: true
			oldText = sourceName + ":\n    enabled: true"
			newFullText = sourceName + ":\n" + newText
			text = strings.Replace(text, oldText, newFullText, 1)
		}
	}

	err = os.WriteFile(configPath, []byte(text), 0644)
	if err != nil {
		return fmt.Errorf("failed to write main config file: %v", err)
	}

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
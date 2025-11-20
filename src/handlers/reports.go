package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

// CombinedReport represents a report file with its metadata
type CombinedReport struct {
	Filename    string `json:"filename"`
	DisplayName string `json:"displayName"`
}

// HandleAPIListCombinedReports returns a list of all combined reports
func HandleAPIListCombinedReports(w http.ResponseWriter, r *http.Request) {
	// Read the combined_reports directory
	entries, err := os.ReadDir("./data/combined_reports/")
	if err != nil {
		http.Error(w, "Failed to read reports directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var reports []CombinedReport

	// Process each file
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
			// Create display name from filename
			displayName := strings.ReplaceAll(entry.Name(), "_", " ")
			displayName = strings.ReplaceAll(displayName, ".html", "")
			// Capitalize first letter of each word (simple implementation)
			words := strings.Fields(displayName)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			displayName = strings.Join(words, " ")

			report := CombinedReport{
				Filename:    entry.Name(),
				DisplayName: displayName,
			}
			reports = append(reports, report)
		}
	}

	// Return JSON response
	response := map[string]interface{}{
		"success": true,
		"reports": reports,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
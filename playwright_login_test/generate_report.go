package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"strings"
	"time"
)

// TestResult represents a single login test result
type TestResult struct {
	Timestamp          string  `json:"timestamp"`
	TestID             string  `json:"test_id"`
	Username           string  `json:"username"`
	Success            int     `json:"success"`
	TotalResponseTime  float64 `json:"total_response_time_ms"`
	APIResponseTime    float64 `json:"api_response_time_ms"`
	UIRenderTime       float64 `json:"ui_render_time_ms"`
	PageLoadTime       float64 `json:"page_load_time_ms"`
	FinalURL           string  `json:"final_url"`
	Error              string  `json:"error"`
}

// TestSummary represents aggregated statistics for a test run
type TestSummary struct {
	TestID            string
	TotalUsers        int
	SuccessfulLogins  int
	FailedLogins      int
	SuccessRate       float64
	AvgTotalTime      float64
	AvgAPITime        float64
	AvgUITime         float64
	AvgPageLoadTime   float64
	MinTotalTime      float64
	MaxTotalTime      float64
	Bottleneck        string
	TestDate          string
	Results           []TestResult
}

func main() {
	fmt.Println("🎯 Playwright Login Test - HTML Report Generator")
	fmt.Println(strings.Repeat("=", 60))

	// Fetch all test IDs from ClickHouse
	testIDs, err := getTestIDs()
	if err != nil {
		fmt.Printf("❌ Error fetching test IDs: %v\n", err)
		os.Exit(1)
	}

	if len(testIDs) == 0 {
		fmt.Println("❌ No test data found in ClickHouse")
		os.Exit(1)
	}

	fmt.Printf("📊 Found %d test run(s)\n\n", len(testIDs))

	// Generate report for each test ID
	for i, testID := range testIDs {
		fmt.Printf("[%d/%d] Generating report for test_id: %s\n", i+1, len(testIDs), testID)
		
		summary, err := getTestSummary(testID)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		filename := fmt.Sprintf("reports/playwright_login_report_%s_%dusers.html", testID, summary.TotalUsers)
		err = generateHTMLReport(summary, filename)
		if err != nil {
			fmt.Printf("  ❌ Error generating report: %v\n", err)
			continue
		}

		fmt.Printf("  ✅ Report saved: %s\n", filename)
	}

	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("✅ All reports generated successfully!")
}

// getTestIDs fetches all unique test IDs from ClickHouse
func getTestIDs() ([]string, error) {
	query := "SELECT DISTINCT test_id FROM monitoring.playwright_login ORDER BY test_id DESC"
	output, err := executeClickHouseQuery(query)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var testIDs []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and kubectl warning messages
		if line != "" && !strings.Contains(line, "Defaulted container") {
			testIDs = append(testIDs, line)
		}
	}

	return testIDs, nil
}

// getTestSummary fetches and aggregates data for a specific test ID
func getTestSummary(testID string) (*TestSummary, error) {
	query := fmt.Sprintf(`
		SELECT 
			timestamp,
			test_id,
			username,
			success,
			total_response_time_ms,
			api_response_time_ms,
			ui_render_time_ms,
			page_load_time_ms,
			final_url,
			error
		FROM monitoring.playwright_login 
		WHERE test_id = '%s'
		ORDER BY total_response_time_ms
		FORMAT JSONEachRow
	`, testID)

	output, err := executeClickHouseQuery(query)
	if err != nil {
		return nil, err
	}

	// Parse JSON results
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var results []TestResult
	for _, line := range lines {
		if line == "" {
			continue
		}
		var result TestResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		results = append(results, result)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for test_id: %s", testID)
	}

	// Calculate summary statistics
	summary := &TestSummary{
		TestID:  testID,
		Results: results,
	}

	var totalTime, apiTime, uiTime, pageLoadTime float64
	minTime := results[0].TotalResponseTime
	maxTime := results[0].TotalResponseTime

	for _, r := range results {
		summary.TotalUsers++
		if r.Success == 1 {
			summary.SuccessfulLogins++
		} else {
			summary.FailedLogins++
		}

		totalTime += r.TotalResponseTime
		apiTime += r.APIResponseTime
		uiTime += r.UIRenderTime
		pageLoadTime += r.PageLoadTime

		if r.TotalResponseTime < minTime {
			minTime = r.TotalResponseTime
		}
		if r.TotalResponseTime > maxTime {
			maxTime = r.TotalResponseTime
		}
	}

	summary.AvgTotalTime = totalTime / float64(summary.TotalUsers)
	summary.AvgAPITime = apiTime / float64(summary.TotalUsers)
	summary.AvgUITime = uiTime / float64(summary.TotalUsers)
	summary.AvgPageLoadTime = pageLoadTime / float64(summary.TotalUsers)
	summary.MinTotalTime = minTime
	summary.MaxTotalTime = maxTime
	summary.SuccessRate = (float64(summary.SuccessfulLogins) / float64(summary.TotalUsers)) * 100

	if summary.AvgUITime > summary.AvgAPITime {
		summary.Bottleneck = "UI Rendering"
	} else {
		summary.Bottleneck = "API Response"
	}

	// Parse test date from first result timestamp
	if len(results) > 0 {
		t, err := time.Parse("2006-01-02 15:04:05.000", results[0].Timestamp)
		if err == nil {
			summary.TestDate = t.Format("January 2, 2006 15:04:05")
		} else {
			summary.TestDate = results[0].Timestamp
		}
	}

	return summary, nil
}

// executeClickHouseQuery executes a query via kubectl
func executeClickHouseQuery(query string) (string, error) {
	cmd := exec.Command("kubectl", "exec", "-i",
		"chi-clickhouse-vusmart-0-0-0",
		"-n", "vsmaps", "--",
		"clickhouse-client",
		"-d", "vusmart",
		"--user", "vusmartmanager",
		"--password", "Vunet#1234",
		"-q", query)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("query failed: %v\nOutput: %s", err, string(output))
	}

	return string(output), nil
}

// generateHTMLReport creates an HTML report file
func generateHTMLReport(summary *TestSummary, filename string) error {
	// Read template from file
	templateContent, err := os.ReadFile("report_template.gohtml")
	if err != nil {
		return fmt.Errorf("failed to read template file: %v", err)
	}
	
	// Create template with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}
	
	tmpl, err := template.New("report").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, summary)
}



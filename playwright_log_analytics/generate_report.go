package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Summary struct {
	AvgTotalTime      float64
	AvgSearchAPITime  float64
	AvgRenderTime     float64
	AvgPageLoadTime   float64
	TotalTests        int
	SuccessfulTests   int
	FailedTests       int
	SuccessRate       float64
}

type FilterStat struct {
	Filter           string
	AvgTotalTime     float64
	AvgSearchAPITime float64
	AvgRenderTime    float64
	SuccessRate      float64
	Count            int
}

type UserStat struct {
	Username         string
	AvgTotalTime     float64
	AvgSearchAPITime float64
	AvgRenderTime    float64
	SuccessRate      float64
	Count            int
}

type DetailedResult struct {
	Timestamp     string
	Username      string
	Filter        string
	TotalTime     float64
	SearchAPITime float64
	RenderTime    float64
	Success       int
}

type ReportData struct {
	TestID          string
	GeneratedAt     string
	Summary         Summary
	FilterStats     []FilterStat
	UserStats       []UserStat
	DetailedResults []DetailedResult
}

// Execute ClickHouse query via kubectl
func execClickHouseQuery(query string) (string, error) {
	cmd := exec.Command("kubectl", "exec", "-i", "chi-clickhouse-vusmart-0-0-0", "-n", "vsmaps", "--",
		"clickhouse-client", "-d", "vusmart", "--user", "vusmartmanager", "--password", "Vunet#1234",
		"--format", "TabSeparated", "-q", query)
	
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("query failed: %v, stderr: %s", err, stderr.String())
	}
	
	return out.String(), nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func main() {
	fmt.Println("📊 Generating Playwright Log Analytics Report...")
	
	// Get the latest test_id
	query := `SELECT test_id FROM monitoring.playwright_log_analytics ORDER BY timestamp DESC LIMIT 1`
	result, err := execClickHouseQuery(query)
	if err != nil {
		log.Fatal("Failed to get test_id:", err)
	}
	
	testID := strings.TrimSpace(result)
	if testID == "" {
		log.Fatal("No test data found in ClickHouse")
	}
	
	fmt.Printf("📋 Test ID: %s\n", testID)
	
	// Get summary statistics
	query = fmt.Sprintf(`
		SELECT
			AVG(total_time_ms),
			AVG(search_api_time_ms),
			AVG(results_render_time_ms),
			AVG(page_load_time_ms),
			COUNT(*),
			SUM(success),
			COUNT(*) - SUM(success),
			(SUM(success) * 100.0 / COUNT(*))
		FROM monitoring.playwright_log_analytics
		WHERE test_id = '%s'
	`, testID)
	
	result, err = execClickHouseQuery(query)
	if err != nil {
		log.Fatal("Failed to get summary:", err)
	}
	
	fields := strings.Split(strings.TrimSpace(result), "\t")
	summary := Summary{
		AvgTotalTime:      parseFloat(fields[0]),
		AvgSearchAPITime:  parseFloat(fields[1]),
		AvgRenderTime:     parseFloat(fields[2]),
		AvgPageLoadTime:   parseFloat(fields[3]),
		TotalTests:        parseInt(fields[4]),
		SuccessfulTests:   parseInt(fields[5]),
		FailedTests:       parseInt(fields[6]),
		SuccessRate:       parseFloat(fields[7]),
	}
	
	// Get filter statistics
	query = fmt.Sprintf(`
		SELECT
			filter,
			AVG(total_time_ms),
			AVG(search_api_time_ms),
			AVG(results_render_time_ms),
			(SUM(success) * 100.0 / COUNT(*)),
			COUNT(*)
		FROM monitoring.playwright_log_analytics
		WHERE test_id = '%s'
		GROUP BY filter
		ORDER BY AVG(total_time_ms) DESC
	`, testID)
	
	result, err = execClickHouseQuery(query)
	if err != nil {
		log.Fatal("Failed to get filter stats:", err)
	}
	
	var filterStats []FilterStat
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 6 {
			filterStats = append(filterStats, FilterStat{
				Filter:           fields[0],
				AvgTotalTime:     parseFloat(fields[1]),
				AvgSearchAPITime: parseFloat(fields[2]),
				AvgRenderTime:    parseFloat(fields[3]),
				SuccessRate:      parseFloat(fields[4]),
				Count:            parseInt(fields[5]),
			})
		}
	}
	
	// Get user statistics
	query = fmt.Sprintf(`
		SELECT
			username,
			AVG(total_time_ms),
			AVG(search_api_time_ms),
			AVG(results_render_time_ms),
			(SUM(success) * 100.0 / COUNT(*)),
			COUNT(*)
		FROM monitoring.playwright_log_analytics
		WHERE test_id = '%s'
		GROUP BY username
		ORDER BY AVG(total_time_ms) DESC
	`, testID)
	
	result, err = execClickHouseQuery(query)
	if err != nil {
		log.Fatal("Failed to get user stats:", err)
	}
	
	var userStats []UserStat
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 6 {
			userStats = append(userStats, UserStat{
				Username:         fields[0],
				AvgTotalTime:     parseFloat(fields[1]),
				AvgSearchAPITime: parseFloat(fields[2]),
				AvgRenderTime:    parseFloat(fields[3]),
				SuccessRate:      parseFloat(fields[4]),
				Count:            parseInt(fields[5]),
			})
		}
	}
	
	// Get detailed results
	query = fmt.Sprintf(`
		SELECT
			formatDateTime(timestamp, '%%Y-%%m-%%d %%H:%%M:%%S'),
			username,
			filter,
			total_time_ms,
			search_api_time_ms,
			results_render_time_ms,
			success
		FROM monitoring.playwright_log_analytics
		WHERE test_id = '%s'
		ORDER BY timestamp DESC
	`, testID)
	
	result, err = execClickHouseQuery(query)
	if err != nil {
		log.Fatal("Failed to get detailed results:", err)
	}
	
	var detailedResults []DetailedResult
	for _, line := range strings.Split(strings.TrimSpace(result), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			detailedResults = append(detailedResults, DetailedResult{
				Timestamp:     fields[0],
				Username:      fields[1],
				Filter:        fields[2],
				TotalTime:     parseFloat(fields[3]),
				SearchAPITime: parseFloat(fields[4]),
				RenderTime:    parseFloat(fields[5]),
				Success:       parseInt(fields[6]),
			})
		}
	}
	
	// Prepare report data
	reportData := ReportData{
		TestID:          testID,
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05"),
		Summary:         summary,
		FilterStats:     filterStats,
		UserStats:       userStats,
		DetailedResults: detailedResults,
	}
	
	// Load template
	tmpl, err := template.ParseFiles("report_template.gohtml")
	if err != nil {
		log.Fatal("Failed to parse template:", err)
	}
	
	// Create reports directory if it doesn't exist
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		log.Fatal("Failed to create reports directory:", err)
	}
	
	// Generate report filename
	reportFile := filepath.Join(reportsDir, fmt.Sprintf("playwright_log_analytics_report_%s.html", testID))
	
	// Create output file
	f, err := os.Create(reportFile)
	if err != nil {
		log.Fatal("Failed to create report file:", err)
	}
	defer f.Close()
	
	// Execute template
	if err := tmpl.Execute(f, reportData); err != nil {
		log.Fatal("Failed to execute template:", err)
	}
	
	fmt.Printf("\n✅ Report generated successfully!\n")
	fmt.Printf("📄 File: %s\n", reportFile)
	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total Tests: %d\n", summary.TotalTests)
	fmt.Printf("   Successful: %d (%.2f%%)\n", summary.SuccessfulTests, summary.SuccessRate)
	fmt.Printf("   Failed: %d\n", summary.FailedTests)
	fmt.Printf("   Avg Total Time: %.2fms\n", summary.AvgTotalTime)
	fmt.Printf("   Avg API Time: %.2fms\n", summary.AvgSearchAPITime)
	fmt.Printf("   Avg Render Time: %.2fms\n", summary.AvgRenderTime)
}

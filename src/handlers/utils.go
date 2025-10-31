package handlers

import (
	"bufio"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

func SendJSONResponse(w http.ResponseWriter, status int, response APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

func ReadLogsFromFile() []map[string]interface{} {
	logFilePath := "logs/vuDataSim.log"
	log.Printf("DEBUG: Attempting to read logs from file: %s", logFilePath)
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Printf("DEBUG: Failed to open log file: %v", err)
		// If log file doesn't exist yet, return empty slice
		return []map[string]interface{}{}
	}
	defer file.Close()
	log.Printf("DEBUG: Successfully opened log file")

	var logs []map[string]interface{}
	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle large log lines
	buf := make([]byte, 0, 1024*1024) // 1MB buffer
	scanner.Buffer(buf, 1024*1024)
	lineCount := 0

	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		var logEntry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
			log.Printf("DEBUG: Failed to parse JSON line %d: %v, line content: %s", lineCount, err, line)
			continue // Skip malformed lines
		}

		// Convert zerolog format to frontend format
		frontendLog := map[string]interface{}{
			"timestamp": ParseZerologTimestamp(logEntry["time"]),
			"node":      GetLogField(logEntry, "node", "System"),
			"module":    GetLogField(logEntry, "module", "System"),
			"message":   GetLogField(logEntry, "message", ""),
			"type":      GetLogType(logEntry),
		}

		logs = append(logs, frontendLog)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("DEBUG: Error scanning file: %v", err)
	}

	log.Printf("DEBUG: Read %d lines from log file, parsed %d valid log entries", lineCount, len(logs))

	// Reverse to show newest first
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs
}

func ParseZerologTimestamp(timeInterface interface{}) string {
	if timeStr, ok := timeInterface.(string); ok {
		if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
			return t.Format("2006-01-02 15:04:05")
		}
	}
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetLogField(entry map[string]interface{}, field, defaultValue string) string {
	if value, ok := entry[field]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func GetLogType(entry map[string]interface{}) string {
	if level, ok := entry["level"]; ok {
		switch level {
		case "error":
			return "error"
		case "warn":
			return "warning"
		case "info":
			return "info"
		case "debug":
			return "info"
		}
	}

	// Check for type field if set by our logging functions
	if logType, ok := entry["type"]; ok {
		if str, ok := logType.(string); ok {
			return str
		}
	}

	return "info"
}

// parseLimitParameter extracts and validates the limit parameter
func ParseLimitParameter(limitStr string) int {
	limit := 100 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	return limit
}

// filterLogs applies node and module filters to the log slice
func FilterLogs(logs []map[string]interface{}, nodeFilter, moduleFilter string) []map[string]interface{} {
	// Early return for no filtering needed
	noNodeFilter := nodeFilter == "" || nodeFilter == "All Nodes"
	noModuleFilter := moduleFilter == "" || moduleFilter == "All Modules"

	if noNodeFilter && noModuleFilter {
		return logs
	}

	if noNodeFilter {
		// Only module filtering needed
		return FilterByModule(logs, moduleFilter)
	}

	// Node filtering needed (with optional module filtering)
	return FilterByNodeAndModule(logs, nodeFilter, moduleFilter, noModuleFilter)
}

// filterByModule filters logs by module only
func FilterByModule(logs []map[string]interface{}, moduleFilter string) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0)
	for _, log := range logs {
		if log["module"] == moduleFilter {
			filtered = append(filtered, log)
		}
	}
	return filtered
}

// filterByNodeAndModule filters logs by node and optionally by module
func FilterByNodeAndModule(logs []map[string]interface{}, nodeFilter, moduleFilter string, noModuleFilter bool) []map[string]interface{} {
	filtered := make([]map[string]interface{}, 0)
	for _, log := range logs {
		if log["node"] == nodeFilter {
			if noModuleFilter || log["module"] == moduleFilter {
				filtered = append(filtered, log)
			}
		}
	}
	return filtered
}

func GetLogs(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	nodeFilter := r.URL.Query().Get("node")
	moduleFilter := r.URL.Query().Get("module")
	limitStr := r.URL.Query().Get("limit")

	// Parse limit parameter
	limitNum := ParseLimitParameter(limitStr)

	// Read logs from the log file
	logs := ReadLogsFromFile()

	// Apply filters and limit in a single pass
	filteredLogs := FilterLogs(logs, nodeFilter, moduleFilter)
	if len(filteredLogs) > limitNum {
		filteredLogs = filteredLogs[:limitNum]
	}

	response := APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"logs":  filteredLogs,
			"total": len(filteredLogs),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (state *AppStates) BroadcastUpdate() {
	data, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	state.Mutex.RLock()
	Clients := make([]*websocket.Conn, 0, len(state.Clients))
	for client := range state.Clients {
		Clients = append(Clients, client)
	}
	state.Mutex.RUnlock()

	for _, client := range Clients {
		go func(c *websocket.Conn) {
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("WebSocket write error: %v", err)
				state.Mutex.Lock()
				delete(state.Clients, c)
				state.Mutex.Unlock()
				c.Close()
			}
		}(client)
	}
}

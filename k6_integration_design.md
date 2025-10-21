# K6 Load Testing Integration Design

## Overview
This document outlines the design for integrating K6 load testing functionality into the vuDataSim Cluster Manager, allowing users to configure global user counts and execute K6 test scripts via API endpoints.

## Current System Analysis

### Existing Components
- **Main Application**: Go-based web server with REST API endpoints
- **Simulation System**: Handles EPS-based load simulation with start/stop functionality
- **K6 Scripts**: External bash script (`k6_all.sh`) that runs multiple K6 dashboard scripts
- **WebSocket**: Real-time updates for connected clients

### Current K6 Script Structure
The `k6_all.sh` script runs multiple K6 scripts with the following format:
```bash
script_path duration virtual_users ramp_up_duration max_duration
```

Example configurations:
- `overall-1.sh 6h 5 5 10` (5 users, 6 hour duration)
- `overall-1.sh 6h 10 10 10` (10 users, 6 hour duration)
- `overall-1.sh 6h 25 25 10` (25 users, 6 hour duration)
- `overall-1.sh 6h 50 50 10` (50 users, 6 hour duration)

## Proposed Solution

### 1. Data Structures

#### K6 Configuration
```go
type K6Config struct {
    GlobalUserCount    int      `json:"globalUserCount"`
    TestDuration       string   `json:"testDuration"` // e.g., "6h", "15m"
    RampUpDuration     int      `json:"rampUpDuration"` // seconds
    MaxDuration        int      `json:"maxDuration"` // seconds
    EnabledScripts     []string `json:"enabledScripts"`
    IntervalBetweenTests int    `json:"intervalBetweenTests"` // seconds
}

type K6Status struct {
    IsRunning         bool      `json:"isRunning"`
    CurrentScript     string    `json:"currentScript,omitempty"`
    StartTime         time.Time `json:"startTime,omitempty"`
    CurrentUserCount  int       `json:"currentUserCount"`
    CompletedScripts  []string  `json:"completedScripts"`
    FailedScripts     []string  `json:"failedScripts"`
    LastError         string    `json:"lastError,omitempty"`
}
```

### 2. API Endpoints

#### K6 Configuration Management
- `GET /api/k6/config` - Get current K6 configuration
- `PUT /api/k6/config` - Update K6 configuration
- `POST /api/k6/config/reset` - Reset to default configuration

#### K6 Test Execution
- `POST /api/k6/start` - Start K6 load testing with current configuration
- `POST /api/k6/stop` - Stop current K6 test execution
- `GET /api/k6/status` - Get current K6 execution status

#### K6 Monitoring
- `GET /api/k6/logs` - Get recent K6 execution logs
- `GET /api/k6/results` - Get test results summary

### 3. Implementation Components

#### K6 Handler (`src/handlers/k6.go`)
```go
type K6Handler struct {
    config     K6Config
    status     K6Status
    mutex      sync.RWMutex
    cmd        *exec.Cmd
}

func (h *K6Handler) StartK6Test(w http.ResponseWriter, r *http.Request)
func (h *K6Handler) StopK6Test(w http.ResponseWriter, r *http.Request)
func (h *K6Handler) GetK6Config(w http.ResponseWriter, r *http.Request)
func (h *K6Handler) UpdateK6Config(w http.ResponseWriter, r *http.Request)
func (h *K6Handler) GetK6Status(w http.ResponseWriter, r *http.Request)
```

#### K6 Script Integration
- **Dynamic Script Generation**: Create a script generator that modifies `k6_all.sh` with configured user count
- **Process Management**: Handle K6 process lifecycle (start, monitor, stop)
- **Log Aggregation**: Collect and parse K6 execution logs

#### Configuration Management
- **Default Configuration**: Provide sensible defaults for all parameters
- **Validation**: Ensure user count and timing parameters are within acceptable ranges
- **Persistence**: Save configuration to file for persistence across restarts

### 4. Integration Points

#### With Existing System
- **AppState Integration**: Add K6 status to global application state
- **WebSocket Broadcasting**: Broadcast K6 status updates to connected clients
- **Logging Integration**: Use existing logger for K6-related events

#### With K6 Scripts
- **Script Template**: Create a template-based approach for generating K6 execution scripts
- **Parameter Injection**: Dynamically inject user count and other parameters into scripts
- **Result Parsing**: Parse K6 output for meaningful metrics and status

### 5. User Interface Integration

#### Frontend Components
- **Configuration Panel**: Web interface for setting global user count and test parameters
- **Status Dashboard**: Real-time display of K6 execution status
- **Results Viewer**: Display test results and logs
- **Control Buttons**: Start/stop test execution

### 6. Security Considerations

#### Execution Safety
- **Path Validation**: Ensure K6 scripts are executed from allowed directories
- **Input Sanitization**: Validate all user-provided parameters
- **Resource Limits**: Implement timeouts and resource monitoring for K6 processes

#### Access Control
- **API Authentication**: Consider adding authentication to K6 endpoints
- **Permission Checks**: Validate user permissions before allowing test execution

### 7. Monitoring and Logging

#### Execution Monitoring
- **Process Health**: Monitor K6 process CPU, memory usage
- **Script Progress**: Track which scripts are currently running
- **Error Detection**: Capture and report script failures

#### Log Management
- **Structured Logging**: Use existing logger with K6-specific context
- **Log Rotation**: Implement log rotation for long-running tests
- **Error Aggregation**: Collect and summarize test failures

### 8. Deployment Considerations

#### File Structure
```
src/
├── handlers/
│   └── k6.go              # K6 integration handler
├── k6_templates/
│   └── k6_all_template.sh # Template for dynamic script generation
└── k6_config.json         # Default configuration file
```

#### Dependencies
- **Process Management**: Use Go's `os/exec` package for script execution
- **File Operations**: Handle template processing and configuration files
- **Concurrency**: Manage concurrent K6 executions safely

### 9. Testing Strategy

#### Unit Tests
- **Configuration Validation**: Test parameter validation logic
- **Script Generation**: Test dynamic script creation
- **Status Management**: Test state management during execution

#### Integration Tests
- **End-to-End Execution**: Test complete K6 execution workflow
- **API Integration**: Test all API endpoints work correctly
- **Error Scenarios**: Test failure handling and recovery

### 10. Future Enhancements

#### Advanced Features
- **Dynamic User Scaling**: Real-time user count adjustment during tests
- **Distributed Execution**: Support for running K6 on multiple nodes
- **Custom Test Scenarios**: Allow users to define custom K6 scripts
- **Performance Analytics**: Detailed performance analysis and reporting

#### Monitoring Integration
- **Metrics Collection**: Integrate with existing metrics collection system
- **Alerting**: Add alerting for test failures or performance issues
- **Historical Analysis**: Store and analyze historical test results

## Implementation Priority

### Phase 1: Core Functionality
1. Basic K6 configuration management
2. Simple script execution with global user count
3. Basic status monitoring

### Phase 2: Enhanced Features
1. Advanced configuration options
2. Detailed logging and error handling
3. WebSocket integration for real-time updates

### Phase 3: Production Ready
1. Comprehensive error handling and recovery
2. Performance optimization
3. Security hardening

This design provides a solid foundation for K6 integration while maintaining compatibility with the existing vuDataSim architecture and allowing for future enhancements.
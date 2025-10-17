Subfile                            |  Approx. LOC  |  Responsibilities / Key Methods                                                                                                 
-----------------------------------+---------------+---------------------------------------------------------------------------------------------------------------------------------
core/manager.js                    |  ~200         |  Entry point creatingVuDataSimManager, initializing submodules, global event bootstrap (DOMContentLoaded,windowtest functions). 
core/api.js                        |  ~150         |  All API interactions such ascallAPI(), error handling, mock responses.                                                         
core/realtime.js                   |  ~100         |  WebSocket setup (setupWebSocket,startRealTimeUpdates,fetchFinalVuDataSimMetrics,displayFinalVuDataSimMetrics).                 
modules/nodes.js                   |  ~350         |  Node CRUD operations, modal management,addNode,updateNode,toggleNode, table refresh, filtering, editing.                       
modules/logs.js                    |  ~150         |  loadLogs,filterLogs,displayLogs, log coloring and styling, random log generation.                                              
modules/metrics/clusterMetrics.js  |  ~300         |  updateDashboardDisplay,updateClusterTableOnly, cluster metric refreshing, SSH checks.                                          
modules/metrics/clickhouse.js      |  ~350         |  ClickHouse metrics: modal control, refresh logic, tables (displayDatabaseMetrics,displaySystemMetrics,displayContainerMetrics).
modules/metrics/podKafka.js        |  ~200         |  displayPodResourceMetrics,displayPodStatusMetrics,displayKafkaTopicMetrics, top pod memory handling.                           
modules/o11ySources.js             |  ~250         |  O11y source management: dropdown, search filter, sync configs, EPS distribution, multi-select logic.                           
modules/binaryControl.js           |  ~300         |  Binary controls:openBinaryControlModal,refreshAllBinaryStatuses,startBinary,stopBinary, SSH output management.                 
ui/elements.js                     |  ~100         |  initializeComponents— central DOM caching layer for all UI elements.                                                           
ui/events.js                       |  ~150         |  bindEventsdefinitions for buttons, modals, filters, and real-time updates.                                                     
ui/notifications.js                |  ~100         |  showNotification,showSyncError,showSyncSuccess, button loading utilities.                                                      






src/
 ├── core/
 │   ├── manager.js
 │   ├── api.js
 │   └── realtime.js
 ├── modules/
 │   ├── nodes.js
 │   ├── logs.js
 │   ├── binaryControl.js
 │   ├── o11ySources.js
 │   └── metrics/
 │       ├── clusterMetrics.js
 │       ├── clickhouse.js
 │       └── podKafka.js
 ├── ui/
 │   ├── elements.js
 │   ├── events.js
 │   └── notifications.js
 └── index.js





# Load Testing Tool - File Structure Refactoring Plan

## Current Issues

### Large Files (>1000 lines)
1. **`src/api_handlers.go`** - 1200 lines
   - Contains all API handlers mixed together
   - Handles simulation, node management, binary control, metrics, ClickHouse, Kafka, and O11y operations
   - Difficult to maintain and extend

2. **`src/node_control/node_manager.go`** - 1384 lines
   - Mixes node management, CLI handling, web server, and API handlers
   - Contains SSH operations, file copying, and configuration management
   - Multiple responsibilities in single file

3. **`src/o11y_source_manager/o11y_source_manager.go`** - 985 lines
   - Handles configuration management, EPS distribution, and remote operations
   - Multiple concerns in single module

### Structural Problems
- **Mixed Responsibilities**: Files contain multiple unrelated functionalities
- **Tight Coupling**: Components are tightly coupled making testing difficult
- **Poor Maintainability**: Large files are hard to navigate and modify
- **No Clear Separation**: API, business logic, and infrastructure code are mixed
- **Configuration Scattered**: Configuration loading and validation spread across files

## Proposed New Structure

### Directory Structure

```
Load_Testing_Tool/
├── src/
│   ├── main.go                          # Entry point (keep as is)
│   ├── middleware.go                    # HTTP middleware (keep as is)
│   ├── websocket.go                     # WebSocket handling (keep as is)
│   ├── utils.go                         # Utility functions (keep as is)
│   ├── ssh_utils.go                     # SSH utilities (keep as is)
│   │
│   ├── handlers/                        # Organized API handlers
│   │   ├── simulation.go               # Simulation start/stop/sync
│   │   ├── dashboard.go                # Dashboard data and health checks
│   │   ├── logs.go                     # Log management and filtering
│   │   ├── node_management.go         # Node CRUD operations
│   │   ├── binary_control.go          # Binary start/stop/status
│   │   ├── metrics.go                 # System and process metrics
│   │   ├── clickhouse.go              # ClickHouse operations
│   │   ├── kafka.go                   # Kafka topic management
│   │   └── o11y_sources.go            # O11y source management
│   │
│   ├── models/                         # Data structures and types
│   │   ├── app_state.go               # Application state management
│   │   ├── config.go                  # Configuration structures
│   │   ├── node.go                    # Node-related types
│   │   ├── response.go                # API response types
│   │   └── types.go                   # Common types and interfaces
│   │
│   ├── services/                      # Business logic layer
│   │   ├── node_manager.go           # Node management business logic
│   │   ├── binary_controller.go      # Binary control business logic
│   │   ├── o11y_manager.go           # O11y source management logic
│   │   ├── clickhouse_client.go      # ClickHouse client wrapper
│   │   ├── kafka_client.go           # Kafka client wrapper
│   │   └── metrics_collector.go      # Metrics collection service
│   │
│   ├── config/                        # Configuration management
│   │   ├── loader.go                 # Configuration loading
│   │   ├── validator.go              # Configuration validation
│   │   ├── manager.go                # Configuration management
│   │   ├── backup.go                 # Configuration backup/versioning
│   │   └── watcher.go                # Configuration hot-reloading
│   │
│   ├── core/                          # Core functionality
│   │   ├── logger.go                 # Logging setup (keep as is)
│   │   ├── constants.go              # Application constants
│   │   └── types.go                  # Core types and interfaces
│   │
│   └── static/                        # Static web assets (moved)
│       ├── index.html
│       ├── js/
│       │   ├── main.js              # Main application logic
│       │   ├── dashboard.js         # Dashboard functionality
│       │   ├── nodes.js             # Node management UI
│       │   ├── metrics.js           # Metrics visualization (existing)
│       │   └── utils.js             # Utility functions
│       ├── css/
│       │   ├── main.css             # Main styles
│       │   ├── dashboard.css        # Dashboard styles
│       │   └── components.css       # Reusable components
│       └── assets/
│           ├── images/              # Icons and logos
│           └── fonts/               # Custom fonts
│
├── conf.d/                           # Service configurations (keep as is)
│   ├── conf.yml
│   ├── Apache/
│   ├── Azure_Firewall/
│   ├── Azure_Redis_Cache/
│   ├── AzureStorageBlob/
│   ├── LinuxMonitor/
│   ├── MongoDB/
│   ├── Mssql/
│   └── ...
│
├── configs/                          # Application configurations (keep as is)
│   ├── config.yaml
│   ├── nodes.yaml
│   ├── max_eps.yaml
│   └── ...
│
├── logs/                            # Log files (keep as is)
├── README.md                        # Documentation (keep as is)
├── go.mod                          # Go modules (keep as is)
└── go.sum                          # Go modules checksum (keep as is)
```

## Detailed File Breakdown

### Handlers Directory

#### `handlers/simulation.go`
- `startSimulation()` - Start simulation with configuration
- `stopSimulation()` - Stop current simulation
- `syncConfiguration()` - Sync external configurations
- Related types: `SimulationConfig`

#### `handlers/dashboard.go`
- `getDashboardData()` - Get current application state
- `healthCheck()` - Application health check
- `serveStatic()` - Serve static files

#### `handlers/logs.go`
- `getLogs()` - Get logs with filtering
- `readLogsFromFile()` - Parse log files
- `parseZerologTimestamp()` - Parse log timestamps
- `getLogField()` - Extract log fields safely
- `getLogType()` - Determine log types

#### `handlers/node_management.go`
- `handleAPINodes()` - GET /api/nodes (list nodes)
- `handleAPINodeActions()` - POST/PUT/DELETE /api/nodes/{name}
- `handleCreateNode()` - Create new node
- `handleUpdateNode()` - Update existing node
- `handleDeleteNode()` - Delete node
- `handleAPIClusterSettings()` - GET/PUT /api/cluster-settings

#### `handlers/binary_control.go`
- `handleAPIGetAllBinaryStatus()` - GET /api/binary/status
- `handleAPIGetBinaryStatus()` - GET /api/binary/status/{node}
- `handleAPIStartBinary()` - POST /api/binary/start/{node}
- `handleAPIStopBinary()` - POST /api/binary/stop/{node}
- `checkSSHConnectivity()` - Check SSH connectivity

#### `handlers/metrics.go`
- `getMetrics()` - GET /api/metrics with time range
- `handleMetricsRequest()` - Process metrics requests
- `handleAPIGetClusterMetrics()` - GET /api/cluster/metrics
- `handleAPIGetProcessMetrics()` - GET /api/process/metrics
- `collectProcessMetricsForNode()` - Collect process metrics via SSH

#### `handlers/clickhouse.go`
- `handleAPIGetClickHouseMetrics()` - GET /api/clickhouse/metrics
- `handleAPIClickHouseHealth()` - GET /api/clickhouse/health

#### `handlers/kafka.go`
- `kafkaHandler.GetTopics()` - GET /api/kafka/topics
- `kafkaHandler.RecreateTopics()` - POST /api/kafka/recreate
- `kafkaHandler.GetTopicStatus()` - GET /api/kafka/status
- `kafkaHandler.DescribeTopic()` - GET /api/kafka/describe/{topic}
- `kafkaHandler.DeleteTopic()` - DELETE /api/kafka/delete/{topic}
- `kafkaHandler.CreateTopic()` - POST /api/kafka/create
- `kafkaHandler.TruncateClickHouseTables()` - POST /api/clickhouse/truncate

#### `handlers/o11y_sources.go`
- `handleAPIGetO11ySources()` - GET /api/o11y/sources
- `handleAPIGetO11ySourceDetails()` - GET /api/o11y/sources/{source}
- `handleAPIDistributeEPS()` - POST /api/o11y/eps/distribute
- `handleAPIGetCurrentEPS()` - GET /api/o11y/eps/current
- `handleAPIEnableO11ySource()` - POST /api/o11y/sources/{source}/enable
- `handleAPIDisableO11ySource()` - POST /api/o11y/sources/{source}/disable
- `handleAPIGetMaxEPSConfig()` - GET /api/o11y/max-eps
- `handleAPIDistributeConfD()` - POST /api/o11y/confd/distribute

### Models Directory

#### `models/app_state.go`
- `AppState` struct - Global application state
- `AppVersion`, `StaticDir`, `Port` constants
- WebSocket client management

#### `models/config.go`
- `APIResponse` - Standard API response format
- Configuration structures from current files
- Validation logic for configurations

#### `models/node.go`
- `NodeConfig` - Node configuration
- `NodeMetrics` - Node metrics data
- `NodesConfig` - Collection of nodes
- `ClusterSettings` - Cluster-wide settings

#### `models/response.go`
- Response types for different API endpoints
- Error response structures
- Success response structures

### Services Directory

#### `services/node_manager.go`
- Core business logic for node management
- Node validation and lifecycle management
- Configuration file operations
- Backup and snapshot management

#### `services/binary_controller.go`
- Binary lifecycle management (start/stop)
- Status monitoring and reporting
- Remote execution abstraction

#### `services/o11y_manager.go`
- O11y source configuration management
- EPS calculation and distribution logic
- Configuration deployment to nodes

#### `services/clickhouse_client.go`
- ClickHouse connection management
- Query execution and metrics collection
- Health check implementation

#### `services/kafka_client.go`
- Kafka client connection and management
- Topic operations (create/delete/describe)
- Consumer group management

#### `services/metrics_collector.go`
- System metrics collection
- Process metrics monitoring
- Metrics aggregation and formatting

### Config Directory

#### `config/loader.go`
- Load configurations from YAML files
- Environment variable substitution
- Configuration file discovery

#### `config/validator.go`
- Validate configuration completeness
- Check for required fields
- Validate configuration relationships

#### `config/manager.go`
- Configuration hot-reloading
- Configuration change notifications
- Configuration versioning

#### `config/backup.go`
- Configuration backup creation
- Backup rotation and cleanup
- Configuration history tracking

#### `config/watcher.go`
- File system watching for config changes
- Automatic reload on file changes
- Change conflict resolution

### Core Directory

#### `core/constants.go`
- Application-wide constants
- Default configuration values
- Timeout and limit constants

#### `core/types.go`
- Common interfaces and types
- Service interfaces for dependency injection
- Error types and handling

### Static Directory

#### JavaScript Organization
- `js/main.js` - Application initialization and routing
- `js/dashboard.js` - Dashboard page functionality
- `js/nodes.js` - Node management interface
- `js/metrics.js` - Metrics visualization (existing)
- `js/utils.js` - Common utility functions

#### CSS Organization
- `css/main.css` - Base styles and layout
- `css/dashboard.css` - Dashboard-specific styles
- `css/components.css` - Reusable UI components

## Migration Strategy

### Phase 1: Preparation
1. Create new directory structure
2. Set up basic models and types
3. Create placeholder files for all new modules

### Phase 2: API Handlers Migration
1. Extract simulation handlers to `handlers/simulation.go`
2. Extract dashboard handlers to `handlers/dashboard.go`
3. Extract log handlers to `handlers/logs.go`
4. Continue with remaining handler categories

### Phase 3: Services Migration
1. Extract business logic from node_manager.go to services
2. Extract binary control logic to services
3. Extract O11y management logic to services

### Phase 4: Configuration Management
1. Implement centralized configuration loading
2. Add configuration validation
3. Implement configuration backup and versioning

### Phase 5: Static Files Organization
1. Move static files to new structure
2. Split JavaScript into logical modules
3. Organize CSS into component-based structure

### Phase 6: Testing and Validation
1. Update all import statements
2. Test each module independently
3. Validate complete application functionality

## Benefits of New Structure

### 1. **Maintainability**
- Smaller, focused files are easier to understand and modify
- Clear separation of concerns makes debugging easier
- Single responsibility principle applied throughout

### 2. **Testability**
- Each module can be unit tested independently
- Dependency injection enables better mocking
- Business logic separated from HTTP concerns

### 3. **Extensibility**
- New API endpoints can be added without affecting existing code
- New service integrations are isolated to specific modules
- Configuration changes don't require touching business logic

### 4. **Code Organization**
- Related functionality is grouped together
- Clear module boundaries prevent tight coupling
- Consistent file and directory naming conventions

### 5. **Developer Experience**
- Easier to find relevant code for specific features
- Faster navigation in IDE with logical structure
- Reduced cognitive load when working on specific features

### 6. **Configuration Management**
- Centralized configuration loading and validation
- Hot-reloading capability for development
- Configuration backup and versioning support

## File Size Reduction

### Before Refactoring
- `api_handlers.go`: 1200 lines → **10+ focused files** (~100-150 lines each)
- `node_manager.go`: 1384 lines → **Multiple focused modules**
- `o11y_source_manager.go`: 985 lines → **Service + handlers separation**

### After Refactoring
- **No file exceeds 500 lines**
- **Average file size: 150-300 lines**
- **Clear module boundaries**
- **Improved code navigation**

## Implementation Priority

### High Priority (Phase 1)
1. API handlers reorganization - Immediate impact on maintainability
2. Models extraction - Foundation for other modules
3. Basic services structure - Core business logic separation

### Medium Priority (Phase 2)
1. Configuration management - Improves deployment flexibility
2. Static files organization - Better frontend maintainability

### Low Priority (Phase 3)
1. Advanced features like hot-reloading - Nice to have
2. Configuration versioning - For production environments

This refactoring will transform the codebase from a monolithic structure to a well-organized, maintainable, and extensible architecture that follows Go best practices and clean architecture principles.
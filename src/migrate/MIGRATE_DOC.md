# Migrate Folder Documentation

## Overview

The `src/migrate/` folder serves as the central deployment repository for all artifacts that are automatically distributed to enabled nodes in the vuDataSim cluster. This folder contains the core binaries and configuration files that are securely copied (via SCP) to all enabled nodes whenever observability sources are configured or nodes are enabled.

## Purpose

The migrate folder acts as the "source of truth" for node deployments, ensuring that:

1. **Consistent Deployment**: All enabled nodes receive identical binaries and configurations
2. **Automated Updates**: Configuration changes are automatically propagated during o11y source operations
3. **Version Control**: All deployment artifacts are tracked in the repository
4. **Rollback Capability**: Previous versions can be restored by reverting the migrate folder contents

## Folder Structure

```
src/migrate/
├── finalvudatasim          # Main application binary
├── conf.d/                 # Configuration directory
│   ├── conf.yml           # Global configuration
│   └── [service]/         # Service-specific configurations
│       ├── conf.yml       # Service configuration
│       └── *.yml         # Service-specific metric definitions
└── MIGRATE_DOC.md         # This documentation
```

## Deployment Process

### Trigger Events

The contents of the migrate folder are automatically deployed to all enabled nodes when:

1. **Node Enablement**: When a node is enabled via `EnableNode()` in `node_control`
2. **O11y Source Configuration**: When observability sources are set up in `o11y_source_manager`
3. **Manual Deployment**: Via explicit calls to `copyFilesToNode()` in `node_control`

### Deployment Steps

1. **Validation**: Check existence of required files (`finalvudatasim`, `conf.d/`)
2. **Directory Creation**: Create remote directories (`BinaryDir`, `ConfDir`) on target node
3. **Binary Deployment**:
   - Copy `finalvudatasim` → `{BinaryDir}/finalvudatasim`
   - Copy `node_metrics_api` → `{BinaryDir}/node_metrics_api` (from separate build location)
4. **Configuration Deployment**: Copy entire `conf.d/` directory → `{ConfDir}/`
5. **Permissions**: Set executable permissions on binaries
6. **Startup**: Launch binaries in background with logging

### Files Deployed

#### Binaries
- **`finalvudatasim`**: Main application binary that handles load testing and data simulation
- **`node_metrics_api`**: Metrics collection API server that exposes system and application metrics

#### Configuration Directory (`conf.d/`)
The `conf.d/` directory contains YAML configuration files organized by service type:

## Service Configurations

### Apache
- **conf.yml**: Apache server connection and authentication settings
- **LogFormat.txt**: Apache log format definitions
- **logs.yml**: Log file monitoring configuration
- **status.yml**: Apache status page metrics collection

### AWS Application Load Balancer (ALB)
- **conf.yml**: AWS credentials and region configuration
- **cloudwatch_aws_application_elb.yml**: CloudWatch metrics for ALB performance

### Azure API Management
- **conf.yml**: Azure subscription and resource configuration
- **backend_request.yml**: Backend service request metrics
- **failure_request.yml**: Failed request tracking
- **successful_request.yml**: Success rate monitoring
- **total_request.yml**: Overall request volume
- **service_capacity.yml**: Capacity utilization metrics

### Azure Firewall
- **conf.yml**: Azure firewall resource configuration
- **application_rules_hit.yml**: Application rule hit counts
- **network_rule_hit_count.yml**: Network rule statistics
- **throughput.yml**: Data throughput metrics
- **snat_port_utilization.yml**: SNAT port usage

### Cisco IOS Switch
- **conf.yml**: SNMP and SSH connection settings
- **cpu.yml**: CPU utilization metrics
- **memory.yml**: Memory usage statistics
- **interface.yml**: Network interface metrics
- **temperature.yml**: Hardware temperature monitoring

### IBM MQ
- **conf.yml**: MQ connection and queue manager settings
- **ChannelMetrics.yml**: Channel performance metrics
- **QMGRMetrics.yml**: Queue manager statistics
- **QueueMetrics.yml**: Queue depth and throughput

### Kubernetes (K8s)
- **conf.yml**: Kubernetes cluster connection settings
- **apiserver.yml**: API server performance metrics
- **controller-manager.yml**: Controller manager metrics
- **scheduler.yml**: Scheduler performance data
- **etcd.yml**: etcd database metrics
- **kubelet.yml**: Node-level kubelet metrics
- **pod.yml**: Pod resource usage
- **node.yml**: Node resource utilization
- **container.yml**: Container-specific metrics

### Linux Monitor
- **conf.yml**: System monitoring configuration
- **cpu.yml**: CPU usage and load averages
- **memory.yml**: RAM and swap utilization
- **diskio.yml**: Disk I/O statistics
- **filesystem.yml**: Filesystem usage and capacity
- **network.yml**: Network interface metrics
- **process.yml**: Process-level monitoring
- **service.yml**: System service status

### MongoDB
- **conf.yml**: MongoDB connection settings
- **db_stats.yml**: Database statistics
- **collection_stats.yml**: Collection-level metrics
- **shard_stats.yml**: Sharding cluster metrics

### MSSQL
- **conf.yml**: SQL Server connection configuration
- **cpu_Stats.yml**: CPU utilization metrics
- **mem_stats.yml**: Memory usage statistics
- **session_details.yml**: Active session monitoring
- **sqlserver_performance_*.yml**: Various performance counters

### Nginx
- **conf.yml**: Nginx server configuration
- **LogFormat.txt**: Log parsing formats
- **logs.yml**: Access log monitoring
- **metrics.yml**: Nginx stub status metrics

### Tomcat
- **conf.yml**: Tomcat server configuration
- **Memory.yml**: JVM memory usage
- **Threading.yml**: Thread pool metrics
- **GarbageCollector.yml**: GC performance statistics
- **GlobalRequestProcessor.yml**: Request processing metrics

## Configuration Update Mechanism

### Dynamic Configuration Updates

When observability sources are configured:

1. **Source Mapping**: O11y sources are mapped to pipeline configurations
2. **Config Generation**: Relevant configuration files are selected based on sources
3. **Deployment Trigger**: `copyFilesToNode()` is called for all enabled nodes
4. **Service Restart**: Binaries are restarted to pick up new configurations

### Configuration File Format

All configuration files use YAML format with the following common structure:

```yaml
# Service connection settings
host: "service.example.com"
port: 8080
username: "monitor"
password: "secret"

# Metric collection settings
interval: 30s
timeout: 10s

# Filters and thresholds
enabled_metrics:
  - cpu_usage
  - memory_usage
thresholds:
  cpu_warning: 80
  memory_critical: 90
```

## Deployment Verification

After deployment, the system performs:

1. **Binary Verification**: Check if binaries exist and are executable
2. **Configuration Validation**: Verify configuration files are present
3. **Service Startup**: Attempt to start services
4. **Health Checks**: Verify services are responding (for metrics API)
5. **Metrics Verification**: Confirm metrics collection is working

## Error Handling and Rollback

### Deployment Failures

If deployment fails:

1. **Partial Cleanup**: Remove partially copied files
2. **Configuration Rollback**: Revert node configuration to previous state
3. **Error Reporting**: Log detailed failure information
4. **Node State**: Mark node as deployment-failed if needed

### Recovery Procedures

1. **Manual Redeployment**: Call `copyFilesToNode()` manually
2. **Configuration Fix**: Update migrate folder contents and redeploy
3. **Node Reset**: Disable and re-enable node to force fresh deployment

## Security Considerations

### File Permissions

- Binaries are set executable (`chmod +x`) during deployment
- Configuration files maintain repository permissions
- SSH keys are used for secure file transfer

### Access Control

- Only enabled nodes receive deployments
- SSH connections use key-based authentication
- No sensitive credentials stored in migrate folder

## Maintenance Procedures

### Updating Binaries

1. Build new binaries in CI/CD pipeline
2. Replace files in `src/migrate/`
3. Commit changes to repository
4. Deploy via node enablement or manual trigger

### Updating Configurations

1. Modify YAML files in `src/migrate/conf.d/`
2. Test configuration syntax
3. Commit changes
4. Trigger deployment to all nodes

### Adding New Services

1. Create new service directory under `conf.d/`
2. Add `conf.yml` and metric definition files
3. Update deployment logic if needed
4. Test deployment and metrics collection

## Monitoring and Troubleshooting

### Deployment Logs

Monitor logs for deployment status:
- Node control logs show deployment progress
- SSH operation logs show file transfer details
- Binary startup logs show service initialization

### Common Issues

1. **Permission Denied**: Check SSH key permissions and remote directory access
2. **Binary Not Found**: Verify build process completed successfully
3. **Configuration Errors**: Validate YAML syntax before deployment
4. **Network Timeouts**: Check SSH connectivity and firewall rules

### Health Checks

Post-deployment verification:
- Binary processes are running (`pgrep`)
- Metrics API responds on configured port
- Configuration files are readable
- Log files are being written

## Integration Points

### Node Control Package

The migrate folder integrates with `src/node_control/`:

- `copyFilesToNode()` handles the actual file transfer
- `EnableNode()` triggers deployment
- SSH operations use configured node credentials

### O11y Source Manager

Integration with `src/o11y_source_manager/`:

- Source configuration triggers config updates
- Pipeline mapping determines which configs to deploy
- Deployment ensures nodes have correct monitoring setup

### Binary Control

Works with `src/bin_control/` for service management:

- Post-deployment startup handled by binary control
- Status monitoring verifies successful deployment
- Automatic restarts ensure configuration pickup

## Future Enhancements

### Planned Improvements

1. **Version Management**: Track deployed versions per node
2. **Incremental Updates**: Only deploy changed files
3. **Configuration Templates**: Dynamic config generation
4. **Rollback Automation**: Automatic rollback on failures
5. **Deployment Orchestration**: Coordinated multi-node updates

### Configuration Validation

1. **Schema Validation**: YAML schema checking before deployment
2. **Dependency Checking**: Verify required services are available
3. **Compatibility Testing**: Ensure config compatibility with binary versions

This migrate folder serves as the critical bridge between the centralized configuration repository and the distributed node infrastructure, ensuring consistent and reliable deployment of monitoring and testing capabilities across the entire vuDataSim cluster.
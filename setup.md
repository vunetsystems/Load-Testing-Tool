# vuDataSim Load Testing Tool Setup Guide

This guide provides detailed instructions for setting up the vuDataSim Load Testing Tool in any cluster environment. The tool is designed to run on a control node that manages distributed load testing across multiple worker nodes via SSH.

## Prerequisites

### System Requirements
- **Go**: Version 1.24 or higher
- **Python**: Version 3.x (for report generation)
- **SSH Access**: Key-based authentication to all worker nodes
- **Cluster Infrastructure**:
  - Kubernetes cluster with the following components:
    - ClickHouse database (vusmart and monitoring databases)
    - Kafka cluster (3 brokers recommended)
    - PostgreSQL database (for additional data storage)
  - Worker nodes for distributed data simulation

### Required Software on Control Node
```bash
# Install Go 1.24+
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install Python 3 and pip
sudo apt-get update
sudo apt-get install python3 python3-pip

# Install required Python packages
pip3 install pandas matplotlib plotly

```

## Cluster Infrastructure Setup

### Expected Kubernetes Pods and Services

The tool expects the following pods to be running in your Kubernetes cluster (namespace: `vsmaps`):

#### Core Database Pods
- `chi-clickhouse-vusmart-0-0-0` - ClickHouse shard 0 replica 0
- `chi-clickhouse-vusmart-0-1-0` - ClickHouse shard 0 replica 1

#### Kafka Broker Pods
- `kafka-cluster-cp-kafka-0` - Kafka broker 0
- `kafka-cluster-cp-kafka-1` - Kafka broker 1
- `kafka-cluster-cp-kafka-2` - Kafka broker 2

#### Load Balancer Pods
- `traefik-*` - Traefik ingress controller pods (multiple instances)

### Database Credentials Setup

#### ClickHouse Users
Create the following users in your ClickHouse cluster:

```sql
-- monitoring_read user for metrics collection
CREATE USER monitoring_read IDENTIFIED WITH sha256_password BY 'StrongP@assword123';
GRANT SELECT ON vusmart.* TO monitoring_read;
GRANT SELECT ON monitoring.* TO monitoring_read;

-- truncate_db user for data cleanup
CREATE USER truncate_db IDENTIFIED WITH sha256_password BY 'StrongP@assword123';
GRANT TRUNCATE ON vusmart.* TO truncate_db;
```

#### PostgreSQL User
Create the following user in your PostgreSQL database:

```sql
-- Load testing tool read user
CREATE USER "Load_Testing_Tool_read_user" WITH PASSWORD 'StrongPassword123';
GRANT SELECT ON ALL TABLES IN SCHEMA public TO "Load_Testing_Tool_read_user";
```

## Configuration Changes

### 1. Update IP Addresses

Edit `src/configs/config.yaml` and update the following IP addresses to match your cluster:

```yaml
network:
  remote_host: "YOUR_CONTROL_NODE_IP"  # IP of the node running vuDataSim
  current_node_ip: "YOUR_CONTROL_NODE_IP"  # Same as above
  port: 8086

clickhouse:
  host: "YOUR_CLICKHOUSE_IP"  # IP of ClickHouse service/pod
  port: 9000

monitoring_db:
  host: "YOUR_CLICKHOUSE_IP"  # Same as above
  port: 9000

truncate_db:
  host: "YOUR_CLICKHOUSE_IP"  # Same as above
  port: 9000

postgres:
  host: "YOUR_POSTGRES_IP"  # IP of PostgreSQL service
  port: 5432
```




## Build and Deployment

### 1. Build the Application

```bash
# Clone the repository
git clone -b o11y_team --single-branch https://github.com/vunetsystems/Load-Testing-Tool.git

cd Load-Testing-Tool

# Build the main manager binary
go build -o vudatasim-manager src/

```

## Starting the Application

### 1. Start the Manager

```bash
# Start the vuDataSim manager
./vudatasim-manager
```

The application will start on `http://YOUR_IP:8086`

### 2. Access the Web Interface

Open your browser and navigate to `http://YOUR_CONTROL_NODE_IP:8086`

### 3. Initial Setup Steps

1. **Authenticate**: Log in using OAuth (configured in .env)
2. **Add Nodes**: Use the web UI to add and enable worker nodes
3. **Configure Sources**: Select observability sources to simulate
4. **Set EPS Targets**: Configure events per second for each source
5. **Deploy Configuration**: Sync configs to all enabled nodes
6. **Start Simulation**: Begin load testing


## Troubleshooting

### Common Issues

1. **ClickHouse Connection Failed**
   ```bash
   # Check ClickHouse pod status
   kubectl get pods -n vsmaps | grep clickhouse

   # Check ClickHouse logs
   kubectl logs -n vsmaps chi-clickhouse-vusmart-0-0-0

   # Verify credentials in config.yaml
   ```

2. **SSH Connection Issues**
   ```bash
   # Test SSH connection to worker nodes
   ssh -i ~/.ssh/id_rsa vunet@WORKER_NODE_IP "echo 'SSH working'"

   # Check SSH key permissions
   chmod 600 ~/.ssh/id_rsa
   ```

3. **Pod Not Found Errors**
   ```bash
   # Verify pod names in your cluster
   kubectl get pods -n vsmaps --show-labels

   # Update pod names in code if different
   # Search for hardcoded pod names in src/ directory
   ```

4. **Database Permission Issues**
   ```sql
   -- Check ClickHouse user permissions
   SHOW GRANTS FOR monitoring_read;

   -- Check PostgreSQL user permissions
   \du "Load_Testing_Tool_read_user"
   ```

### Logs and Debugging

```bash
# Application logs
tail -f logs/vuDataSim.log

# Worker node logs
ssh vunet@WORKER_NODE_IP "tail -f /home/vunet/vuDataSim_bin/logs/*.log"

# ClickHouse query logs
kubectl logs -n vsmaps chi-clickhouse-vusmart-0-0-0 | grep ERROR

# Kafka broker logs
kubectl logs -n vsmaps kafka-cluster-cp-kafka-0
```

## Scaling and Performance

### Cluster Sizing Recommendations

- **Small Cluster**: 3 worker nodes, 5K-10K EPS total
- **Medium Cluster**: 5-10 worker nodes, 10K-50K EPS total
- **Large Cluster**: 10+ worker nodes, 50K+ EPS total

### Monitoring Resource Usage

```bash
# Monitor pod resource usage
kubectl top pods -n vsmaps

# Check node capacity
kubectl describe nodes

# Monitor ClickHouse performance
kubectl exec -n vsmaps chi-clickhouse-vusmart-0-0-0 -- clickhouse-client --query "SELECT * FROM system.metrics"
```

## Backup and Recovery

### Configuration Backup

```bash
# Backup configurations
cp src/configs/*.yaml backups/
cp .env backups/

# Backup database
sqlite3 data/vudatasim.db .dump > backups/database_backup.sql
```

### Recovery Steps

1. Restore configurations from backups
2. Rebuild binaries: `go build -o vudatasim-manager src/main.go`
3. Restart application: `./vudatasim-manager`
4. Re-enable nodes through web UI
5. Verify cluster connectivity

## Security Considerations

1. **SSH Keys**: Use strong SSH keys, rotate regularly
2. **Database Credentials**: Use strong passwords, consider certificate-based auth
3. **Network Security**: Restrict access to control node and databases
4. **OAuth Configuration**: Properly configure OAuth for web UI access
5. **Firewall Rules**: Allow necessary ports (8086 for web UI, 22 for SSH, DB ports)

This setup guide ensures your vuDataSim Load Testing Tool is properly configured and running in any cluster environment with all necessary components and credentials in place.
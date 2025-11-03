package clickhouse

import (
	"context"
	"fmt"
	"time"
	"vuDataSim/src/logger"
)

// KafkaNetworkData represents a single network metrics data point for Kafka hosts
type KafkaNetworkData struct {
	Timestamp     time.Time `json:"timestamp"`
	HostName      string    `json:"host_name"`
	InterfaceName string    `json:"interface_name"`
	BytesSentSec  float64   `json:"bytes_sent_sec"`
	BytesRecvSec  float64   `json:"bytes_recv_sec"`
	PacketsSentSec float64  `json:"packets_sent_sec"`
	PacketsRecvSec float64  `json:"packets_recv_sec"`
	DropOutSec    float64   `json:"drop_out_sec"`
	DropInSec     float64   `json:"drop_in_sec"`
}

// GetKafkaNetworkData fetches Kafka network metrics data from the last 6 hours
func GetKafkaNetworkData(ctx context.Context) ([]KafkaNetworkData, error) {
	if monitoringDBClient == nil {
		return nil, fmt.Errorf("Monitoring ClickHouse client not initialized")
	}

	query := `
SELECT
    toTimeZone(timestamp, 'Asia/Kolkata') AS "Timestamp (IST)",
    host AS "Host Name",
    interface AS "Interface Name",
    (bytes_sent - previous_bytes_sent) / 60 AS "Bytes sent / sec",
    (bytes_recv - previous_bytes_recv) / 60 AS "Bytes recv / sec",
    (packets_sent - previous_packets_sent) / 60 AS "Packets sent / sec",
    (packets_recv - previous_packets_recv) / 60 AS "Packets recv / sec",
    (drop_out - previous_drop_out) / 60 AS "Drop out / sec",
    (drop_in - previous_drop_in) / 60 AS "Drop in / sec"
FROM
(
    SELECT
        timestamp,
        host,
        interface,
        bytes_sent,
        bytes_recv,
        packets_sent,
        packets_recv,
        drop_out,
        drop_in,
        lagInFrame(bytes_sent, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_bytes_sent,
        lagInFrame(bytes_recv, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_bytes_recv,
        lagInFrame(packets_sent, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_packets_sent,
        lagInFrame(packets_recv, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_packets_recv,
        lagInFrame(drop_out, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_drop_out,
        lagInFrame(drop_in, 1) OVER (PARTITION BY host, interface ORDER BY timestamp ASC) AS previous_drop_in
    FROM net
    WHERE (timestamp >= (toTimeZone(now(), 'Asia/Kolkata') - toIntervalHour(6))) AND (interface LIKE 'eth0')
) AS out
ORDER BY "Timestamp (IST)" DESC
`

	rows, err := monitoringDBClient.Client.Query(ctx, query)
	if err != nil {
		logger.LogError("System", "ClickHouse", fmt.Sprintf("Failed to query Kafka network data: %v", err))
		return nil, fmt.Errorf("failed to query Kafka network data: %v", err)
	}
	defer rows.Close()

	var results []KafkaNetworkData
	for rows.Next() {
		var result KafkaNetworkData
		err := rows.Scan(
			&result.Timestamp,
			&result.HostName,
			&result.InterfaceName,
			&result.BytesSentSec,
			&result.BytesRecvSec,
			&result.PacketsSentSec,
			&result.PacketsRecvSec,
			&result.DropOutSec,
			&result.DropInSec,
		)
		if err != nil {
			logger.LogWarning("System", "ClickHouse", fmt.Sprintf("Failed to scan Kafka network row: %v", err))
			continue
		}
		results = append(results, result)
	}

	logger.LogWithNode("System", "ClickHouse", fmt.Sprintf("Fetched %d Kafka network data points", len(results)), "info")
	return results, nil
}
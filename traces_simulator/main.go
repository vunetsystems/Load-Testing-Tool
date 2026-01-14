package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/IBM/sarama"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)


/* ===================== CONFIG ===================== */

const (
	namespace = "vsmaps"

	// POD NAMES
	kafkaPodName      = "kafka-cluster-cp-kafka-0"
	clickhousePodName = "chi-clickhouse-vusmart-0-0-0"
	tracesPodPrefix   = "traces-1-1-1-1-"

	// KAFKA
	kafkaTopic = "traces-input"
	kafkaPort  = "9094"

	// CLICKHOUSE
	clickhouseDB   = "vusmart"
	clickhousePort = "9000"

	traceBinary = "./otel-trace"
	configFile  = "config.json"
)

const (
	clickhouseDBUser = "truncate_db"
	clickhouseDBPass = "StrongP@assword123"
)

var (
	epsFlag      = flag.Int("eps", 0, "Total EPS to generate")
	minutesFlag  = flag.Int("minutes", 0, "Duration to run (in minutes)")
	testNameFlag = flag.String("test-name", "", "Test name")
)


/* ===================== CLICKHOUSE TABLES ===================== */

var clickhouseTables = []string{
	"vtraces_apm_span_data",
	"vtraces_metadata_data",
	"vtraces_metadata_data_v1",
	"vtraces_rum_json_data",
	"vtraces_rum_transactions_1min_table_data",
	"vtraces_service_node_map_1min_table_data",
	"vtraces_service_request_metrics_table_data_filters_data",
	"vtraces_service_request_metrics_transactions_1min_table_data",
	"vtraces_service_request_metrics_transactions_1min_table_data_quantile_data",
	"vmetrics_apm_span_data",
	"vmetrics_apm_span_presampling_summary_data",
}

/* ===================== MAIN ===================== */

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("========== TRACE RUNNER START ==========")

	flag.Parse()

	if *epsFlag <= 0 || *minutesFlag <= 0 || *testNameFlag == "" {
		log.Fatalf("Usage: --eps <number> --minutes <number> --test-name <name>")
	}

	eps := *epsFlag
	minutes := *minutesFlag
	testName := *testNameFlag

	testID := uuid.NewString()

	configEps := eps / 10
	if configEps == 0 {
		configEps = 1
	}

	// 1. Recreate Kafka topic
	mustOK(recreateKafkaTopic())

	// 2. Recycle traces pod
	tracesIP := must(recycleTracesPod())

	// 3. Update config.json
	mustOK(updateConfig(configFile, tracesIP, configEps))

	// 4. Truncate ClickHouse
	clickhouseNode, err := getClickHouseNodeName()
	mustOK(err)
	mustOK(truncateClickHouse(clickhouseNode + ":" + clickhousePort))

	// 5. Run trace generator
	startTime := time.Now().UTC()
	mustOK(runTraceBinary(minutes))
	endTime := time.Now().UTC()

	startTimeStr := startTime.Format("2006-01-02 15:04:05.99999999+00:00")
	endTimeStr := endTime.Format("2006-01-02 15:04:05.99999999+00:00")


	// 6. Insert test run record
	mustOK(insertTestRun(
		testID,
		testName,
		eps,
		startTimeStr,
		endTimeStr,
		minutes*60,
	))

	log.Println("========== TRACE RUNNER COMPLETED ==========")
}


/* ===================== GENERIC ===================== */

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	return v
}

func mustOK(err error) {
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

/* ===================== COMMAND RUNNER ===================== */

func runCmd(name string, args ...string) (string, error) {
	log.Printf("CMD: %s %s", name, strings.Join(args, " "))

	cmd := exec.Command(name, args...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if stdout.Len() > 0 {
		log.Printf("STDOUT:\n%s", stdout.String())
	}
	if stderr.Len() > 0 {
		log.Printf("STDERR:\n%s", stderr.String())
	}

	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

/* ===================== KUBERNETES ===================== */

func recycleTracesPod() (string, error) {
	log.Println("K8s: deleting traces pod(s)")

	_, err := runCmd(
		"sh", "-c",
		fmt.Sprintf(
			`kubectl -n %s get pods -o name | grep %s | xargs -r kubectl -n %s delete`,
			namespace, tracesPodPrefix, namespace,
		),
	)
	if err != nil {
		return "", err
	}

	log.Println("K8s: waiting for new traces pod")

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for traces pod")
		default:
		}

		out, _ := runCmd(
			"sh", "-c",
			fmt.Sprintf(
				`kubectl -n %s get pods -o wide | grep %s | grep Running`,
				namespace, tracesPodPrefix,
			),
		)

		lines := strings.Split(out, "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				ip := fields[5]
				log.Printf("K8s: new traces pod IP = %s", ip)
				return ip, nil
			}
		}

		time.Sleep(3 * time.Second)
	}
}

/* ===================== KAFKA (DESCRIBE → DELETE → RECREATE) ===================== */

func recreateKafkaTopic() error {
	nodeName, err := getKafkaNodeName()
	if err != nil {
		return err
	}

	broker := nodeName + ":" + kafkaPort
	log.Printf("Kafka: connecting via %s", broker)

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_6_0_0

	cfg.Net.TLS.Enable = true
	cfg.Net.TLS.Config = &tls.Config{
		InsecureSkipVerify: true,
	}

	admin, err := sarama.NewClusterAdmin([]string{broker}, cfg)
	if err != nil {
		return fmt.Errorf("kafka admin init failed: %w", err)
	}
	defer admin.Close()

	// rest of your function unchanged

	// 1. Describe topic
	log.Printf("Kafka: describing topic %s", kafkaTopic)

	meta, err := admin.DescribeTopics([]string{kafkaTopic})
	if err != nil {
		return err
	}
	if len(meta) == 0 {
		return fmt.Errorf("topic %s not found", kafkaTopic)
	}

	partitions := len(meta[0].Partitions)
	if partitions == 0 {
		return fmt.Errorf("topic %s has no partitions", kafkaTopic)
	}
	replication := int16(len(meta[0].Partitions[0].Replicas))

	log.Printf(
		"Kafka: captured metadata partitions=%d replication=%d",
		partitions, replication,
	)

	// 2. Fetch topic configs
	cfgs, err := admin.DescribeConfig(
		sarama.ConfigResource{
			Type: sarama.TopicResource,
			Name: kafkaTopic,
		},
	)
	if err != nil {
		return err
	}

	configEntries := make(map[string]*string)
	for _, c := range cfgs {
		if c.Value != "" && !c.ReadOnly {
			val := c.Value
			configEntries[c.Name] = &val
		}
	}

	// 3. Delete topic
	log.Printf("Kafka: deleting topic %s", kafkaTopic)
	if err := admin.DeleteTopic(kafkaTopic); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	// 4. Recreate topic with same details
	log.Printf("Kafka: recreating topic %s", kafkaTopic)
	err = admin.CreateTopic(
		kafkaTopic,
		&sarama.TopicDetail{
			NumPartitions:     int32(partitions),
			ReplicationFactor: replication,
			ConfigEntries:     configEntries,
		},
		false,
	)
	if err != nil {
		return err
	}

	log.Println("Kafka: topic recreated successfully")
	return nil
}

/* ===================== CLICKHOUSE ===================== */

func truncateClickHouse(addr string) error {
	log.Printf("ClickHouse: connecting to %s", addr)

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: clickhouseDB,
			Username: clickhouseDBUser,
			Password: clickhouseDBPass,
		},
		DialTimeout: 10 * time.Second,
	})

	if err != nil {
		return err
	}

	ctx := context.Background()

	for _, table := range clickhouseTables {
		query := fmt.Sprintf("TRUNCATE TABLE %s.%s", clickhouseDB, table)
		log.Printf("ClickHouse: %s", query)

		if err := conn.Exec(ctx, query); err != nil {
			return fmt.Errorf("truncate failed for %s: %w", table, err)
		}
	}

	log.Println("ClickHouse: all tables truncated")
	return nil
}

/* ===================== CONFIG UPDATE ===================== */

func updateConfig(path, newCollectorIP string, newEps int) error {
	log.Printf("Config: updating collectorEndpoint IP → %s", newCollectorIP)
	log.Printf("Config: updating eps → %d", newEps)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	// ---- Update collectorEndpoint (IP only) ----
	raw, ok := cfg["collectorEndpoint"].(string)
	if !ok || raw == "" {
		return fmt.Errorf("collectorEndpoint missing or invalid")
	}

	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid collectorEndpoint format: %s", raw)
	}

	port := parts[1]
	cfg["collectorEndpoint"] = fmt.Sprintf("%s:%s", newCollectorIP, port)

	// ---- Update EPS ----
	cfg["eps"] = newEps

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

/* ===================== TRACE SIMULATOR ===================== */

func runTraceBinary(minutes int) error {
	log.Println("TraceGen: starting simulator")

	cmd := exec.Command(traceBinary)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// Feed duration to the binary
	go func() {
		defer stdin.Close()
		fmt.Fprintf(stdin, "%d\n", minutes)
	}()

	return cmd.Wait()
}

func getKafkaNodeName() (string, error) {
	log.Println("K8s: fetching Kafka node name")

	out, err := runCmd(
		"kubectl",
		"-n", namespace,
		"get", "pod", kafkaPodName,
		"-o", "jsonpath={.spec.nodeName}",
	)
	if err != nil {
		return "", fmt.Errorf("failed to get kafka node name: %w", err)
	}

	if out == "" {
		return "", fmt.Errorf("empty node name returned for kafka pod")
	}
	print(out)
	log.Printf("K8s: Kafka pod %s is running on node %s", kafkaPodName, out)
	return out, nil
}

func getClickHouseNodeName() (string, error) {
	log.Println("K8s: fetching ClickHouse node name")

	out, err := runCmd(
		"kubectl",
		"-n", namespace,
		"get", "pod", clickhousePodName,
		"-o", "jsonpath={.spec.nodeName}",
	)
	if err != nil {
		return "", fmt.Errorf("failed to get clickhouse node name: %w", err)
	}

	if out == "" {
		return "", fmt.Errorf("empty node name returned for clickhouse pod")
	}

	log.Printf(
		"K8s: ClickHouse pod %s is running on node %s",
		clickhousePodName, out,
	)
	return out, nil
}


func insertTestRun(
	testID string,
	testName string,
	targetEPS int,
	startTime string,
	endTime string,
	timeoutSeconds int,
) error {

	dbPath := "../data/vudatasim.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	o11ySources := []string{"traces"}

	o11yJSON, err := json.Marshal(o11ySources)
	if err != nil {
		return err
	}


	query := `
	INSERT INTO test_runs (
		test_id,
		test_name,
		target_eps,
		start_time,
		end_time,
		o11y_sources,
		timeout_seconds,
		status,
		kafka_summary_generated
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.Exec(
		query,
		testID,
		testName,
		targetEPS,
		startTime,
		endTime,
		string(o11yJSON),
		timeoutSeconds,
		"completed",
		0,
	)

	if err != nil {
		return fmt.Errorf("failed to insert test_run: %w", err)
	}

	log.Printf("DB: test_run inserted (test_id=%s)", testID)
	return nil
}

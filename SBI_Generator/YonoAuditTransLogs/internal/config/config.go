package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete YAML configuration
type Config struct {
	MessageType  string             `yaml:"message_type"` // "json_yono" or "access_log"
	Execution    ExecutionConfig    `yaml:"execution"`
	Distribution DistributionConfig `yaml:"distribution"`
	Session      SessionConfig      `yaml:"session"`
	Templates    TemplatesConfig    `yaml:"templates"`
	AccessLog    AccessLogConfig    `yaml:"access_log"`
	EIS          EISConfig          `yaml:"eis"`
	UserIDs      UserIDsConfig      `yaml:"user_ids"`
	IDGeneration IDGenerationConfig `yaml:"id_generation"`
	Kafka        KafkaConfig        `yaml:"kafka"`
	Concurrency  ConcurrencyConfig  `yaml:"concurrency"`
	Logging      LoggingConfig      `yaml:"logging"`
	Monitoring   MonitoringConfig   `yaml:"monitoring"`
}

// ExecutionConfig controls execution mode and throughput
type ExecutionConfig struct {
	Mode     string `yaml:"mode"`     // "duration" or "count"
	Duration string `yaml:"duration"` // Duration string (e.g., "5m")
	Count    int    `yaml:"count"`    // Total messages to generate
	EPS      int    `yaml:"eps"`      // Events per second target
}

// DistributionConfig defines message type distribution
type DistributionConfig struct {
	ErrorOnlyPercent int `yaml:"error_only_percent"` // % of messages with only error
	TransOnlyPercent int `yaml:"trans_only_percent"` // % of messages with only trans
	BothPercent      int `yaml:"both_percent"`       // % of messages with both
}

// SessionConfig controls session ID management
type SessionConfig struct {
	RotationInterval string `yaml:"rotation_interval"` // How often to rotate session ID
	SessionIDPrefix  string `yaml:"session_id_prefix"` // Prefix for session IDs
}

// TemplatesConfig contains message templates
type TemplatesConfig struct {
	Error       ErrorTemplateConfig       `yaml:"error"`
	Transaction TransactionTemplateConfig `yaml:"transaction"`
}

// ErrorTemplateConfig contains error message templates
type ErrorTemplateConfig struct {
	ErrorCodes         []string `yaml:"error_codes"`
	ErrorTypes         []string `yaml:"error_types"`
	ErrorDescriptions  []string `yaml:"error_descriptions"`
	ErrorDetails       []string `yaml:"error_details"`
	CreatedBy          string   `yaml:"created_by"`
	WrapInMessage      bool     `yaml:"wrap_in_message"`
}

// TransactionTemplateConfig contains transaction message templates
type TransactionTemplateConfig struct {
	Statuses                []StatusWeight `yaml:"statuses"`
	UserTypes               []string       `yaml:"user_types"`
	CommandIDs              []string       `yaml:"command_ids"`
	ChannelIDs              []int          `yaml:"channel_ids"`
	UserRelationshipNumbers []string       `yaml:"user_relationship_numbers"`
	BizReqInputs            []string       `yaml:"biz_req_inputs"`
	BizRespOutputs          []string       `yaml:"biz_resp_outputs"`
	CreatedBy               string         `yaml:"created_by"`
	WrapInMessage           bool           `yaml:"wrap_in_message"`
}

// AccessLogConfig contains access log message templates
type AccessLogConfig struct {
	PodNames       []string       `yaml:"pod_names"`
	LogPaths       []string       `yaml:"log_paths"`
	LogNames       []string       `yaml:"log_names"`
	ChannelIDs     []int          `yaml:"channel_ids"`
	ChannelVersion string         `yaml:"channel_version"`
	APIUrls        []string       `yaml:"api_urls"`
	HttpStatuses   []StatusWeight `yaml:"http_statuses"`
	WrapInMessage  bool           `yaml:"wrap_in_message"`
}

// EISConfig contains EIS message templates
type EISConfig struct {
	ServiceIDs    []string `yaml:"service_ids"`
	ErrorCodes    []string `yaml:"error_codes"`
	ErrorMessages []string `yaml:"error_messages"`
	SystemNames   []string `yaml:"system_names"`
	APIUrls       []string `yaml:"api_urls"`
	CreatedBy     string   `yaml:"created_by"`
	WrapInMessage bool     `yaml:"wrap_in_message"`
}

// StatusWeight represents weighted transaction status
type StatusWeight struct {
	Weight int    `yaml:"weight"`
	Value  string `yaml:"value"`
}

// UserIDsConfig controls user ID generation
type UserIDsConfig struct {
	Mode      string  `yaml:"mode"`       // "fixed" or "range"
	FixedList []int64 `yaml:"fixed_list"` // List of fixed user IDs
	RangeMin  int64   `yaml:"range_min"`  // Min for random range
	RangeMax  int64   `yaml:"range_max"`  // Max for random range
}

// IDGenerationConfig controls ID generation patterns
type IDGenerationConfig struct {
	TransactionIDPattern string `yaml:"transaction_id_pattern"` // Pattern for transaction ID
	RequestNoLength      int    `yaml:"request_no_length"`      // Length of request number
	TraceIDFormat        string `yaml:"trace_id_format"`        // "hex" or "uuid"
	TraceIDLength        int    `yaml:"trace_id_length"`        // Length for hex format
}

// KafkaConfig contains Kafka producer settings
type KafkaConfig struct {
	Brokers        []string         `yaml:"brokers"`
	Topic          string           `yaml:"topic"`
	AccessLogTopic string           `yaml:"access_log_topic"`
	EISTopic       string           `yaml:"eis_topic"`
	Producer       ProducerConfig   `yaml:"producer"`
	Connection     ConnectionConfig `yaml:"connection"`
}

// ProducerConfig contains Kafka producer settings
type ProducerConfig struct {
	NumProducers     int    `yaml:"num_producers"`
	RequiredAcks     int    `yaml:"required_acks"`
	Compression      string `yaml:"compression"`
	MaxMessageBytes  int    `yaml:"max_message_bytes"`
	FlushFrequency   string `yaml:"flush_frequency"`
	FlushMessages    int    `yaml:"flush_messages"`
	Idempotent       bool   `yaml:"idempotent"`
	RetryMax         int    `yaml:"retry_max"`
	RetryBackoff     string `yaml:"retry_backoff"`
	EnableKey        bool   `yaml:"enable_key"`
}

// ConnectionConfig contains Kafka connection settings
type ConnectionConfig struct {
	Timeout            string `yaml:"timeout"`
	KeepAlive          string `yaml:"keep_alive"`
	EnableTLS          bool   `yaml:"enable_tls"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// ConcurrencyConfig controls concurrency parameters
type ConcurrencyConfig struct {
	WorkerPoolSize   int `yaml:"worker_pool_size"`
	BufferSize       int `yaml:"buffer_size"`
	RateLimiterBurst int `yaml:"rate_limiter_burst"`
}

// LoggingConfig controls logging behavior
type LoggingConfig struct {
	Level           string `yaml:"level"`
	Format          string `yaml:"format"`
	MetricsInterval string `yaml:"metrics_interval"`
}

// MonitoringConfig controls monitoring settings
type MonitoringConfig struct {
	EnableMetrics bool `yaml:"enable_metrics"`
	MetricsPort   int  `yaml:"metrics_port"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDuration parses the duration string
func (e *ExecutionConfig) GetDuration() (time.Duration, error) {
	return time.ParseDuration(e.Duration)
}

// GetSessionRotationInterval parses the session rotation interval
func (s *SessionConfig) GetRotationInterval() (time.Duration, error) {
	return time.ParseDuration(s.RotationInterval)
}

// GetFlushFrequency parses the flush frequency
func (p *ProducerConfig) GetFlushFrequency() (time.Duration, error) {
	return time.ParseDuration(p.FlushFrequency)
}

// GetRetryBackoff parses the retry backoff duration
func (p *ProducerConfig) GetRetryBackoff() (time.Duration, error) {
	return time.ParseDuration(p.RetryBackoff)
}

// GetTimeout parses the connection timeout
func (c *ConnectionConfig) GetTimeout() (time.Duration, error) {
	return time.ParseDuration(c.Timeout)
}

// GetKeepAlive parses the keep alive duration
func (c *ConnectionConfig) GetKeepAlive() (time.Duration, error) {
	return time.ParseDuration(c.KeepAlive)
}

// GetMetricsInterval parses the metrics interval
func (l *LoggingConfig) GetMetricsInterval() (time.Duration, error) {
	return time.ParseDuration(l.MetricsInterval)
}

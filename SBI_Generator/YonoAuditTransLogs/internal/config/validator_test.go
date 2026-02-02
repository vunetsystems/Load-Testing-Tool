package config

import (
	"testing"
)

func TestValidateDistribution(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid distribution",
			config: Config{
				Distribution: DistributionConfig{
					ErrorOnlyPercent: 10,
					TransOnlyPercent: 60,
					BothPercent:      30,
				},
				Execution: ExecutionConfig{
					Mode: "duration",
					Duration: "5m",
					EPS: 1000,
				},
				Session: SessionConfig{
					RotationInterval: "30s",
				},
				Kafka: KafkaConfig{
					Brokers: []string{"localhost:9092"},
					Topic: "test",
					Producer: ProducerConfig{
						RequiredAcks: 1,
						Compression: "snappy",
						FlushFrequency: "100ms",
						RetryBackoff: "100ms",
						NumProducers: 1,
					},
					Connection: ConnectionConfig{
						Timeout: "10s",
						KeepAlive: "30s",
					},
				},
				UserIDs: UserIDsConfig{
					Mode: "fixed",
					FixedList: []int64{123456},
				},
				IDGeneration: IDGenerationConfig{
					TraceIDFormat: "hex",
				},
				Templates: TemplatesConfig{
					Error: ErrorTemplateConfig{
						ErrorCodes: []string{"ERR001"},
					},
					Transaction: TransactionTemplateConfig{
						CommandIDs: []string{"CMD001"},
						Statuses: []StatusWeight{
							{Weight: 100, Value: "success"},
						},
					},
				},
				Concurrency: ConcurrencyConfig{
					WorkerPoolSize: 10,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid distribution sum",
			config: Config{
				Distribution: DistributionConfig{
					ErrorOnlyPercent: 10,
					TransOnlyPercent: 60,
					BothPercent:      20, // Sum is 90, not 100
				},
				Execution: ExecutionConfig{
					Mode: "duration",
					Duration: "5m",
					EPS: 1000,
				},
			},
			wantErr: true,
		},
		{
			name: "Invalid execution mode",
			config: Config{
				Distribution: DistributionConfig{
					ErrorOnlyPercent: 0,
					TransOnlyPercent: 70,
					BothPercent:      30,
				},
				Execution: ExecutionConfig{
					Mode: "invalid",
					EPS: 1000,
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

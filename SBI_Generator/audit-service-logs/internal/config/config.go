package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Execution struct {
		TargetEPS            int    `yaml:"targetEPS"`
		Duration             string `yaml:"duration"`
		Workers              int    `yaml:"workers"`
		NullTimestampPercent int    `yaml:"nullTimestampPercent"`
	} `yaml:"execution"`

	MessageTemplate struct {
		MsgId    interface{} `yaml:"msgId"`
		SrvcRqst interface{} `yaml:"srvcRqst"`
		SrvcRspn interface{} `yaml:"srvcRspn"`
		UsrId    interface{} `yaml:"usrId"`
		CrtdBy   string      `yaml:"crtdBy"`
	} `yaml:"messageTemplate"`

	ErrorTemplate struct {
		TrnsSts    string `yaml:"trnsSts"`
		ErrCd      string `yaml:"errCd"`
		ErrMsg     string `yaml:"errMsg"`
		SystemName string `yaml:"systemName"`
	} `yaml:"errorTemplate"`

	Kafka struct {
		Namespace string `yaml:"namespace"`
		PodName   string `yaml:"podName"`
		Port      string `yaml:"port"`
		Topic     string `yaml:"topic"`
	} `yaml:"kafka"`

	Services map[string]struct {
		ApiUrl string `yaml:"apiUrl"`
	} `yaml:"services"`
}

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

func (c *Config) GetDuration() (time.Duration, error) {
	return time.ParseDuration(c.Execution.Duration)
}

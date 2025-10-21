package handlers

import (
	"sync"
	"time"
	"vuDataSim/src/bin_control"
	"vuDataSim/src/clickhouse"
	"vuDataSim/src/node_control"
	"vuDataSim/src/o11y_source_manager"

	"github.com/gorilla/websocket"
)

type ProcessMetrics struct {
	NodeID     string    `json:"nodeId"`
	Running    bool      `json:"running"`
	PID        int       `json:"pid,omitempty"`
	StartTime  string    `json:"start_time,omitempty"`
	CPUPercent float64   `json:"cpu_percent,omitempty"`
	MemMB      float64   `json:"mem_mb,omitempty"`
	Cmdline    string    `json:"cmdline,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Error      string    `json:"error,omitempty"`
}

type SSHStatus struct {
	NodeName    string `json:"nodeName"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	LastChecked string `json:"lastChecked"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SimulationConfig struct {
	Profile          string `json:"profile"`
	TargetEPS        int    `json:"targetEps"`
	TargetKafka      int    `json:"targetKafka"`
	TargetClickHouse int    `json:"targetClickHouse"`
}

type AppStates struct {
	IsSimulationRunning bool                                 `json:"isSimulationRunning"`
	CurrentProfile      string                               `json:"currentProfile"`
	TargetEPS           int                                  `json:"targetEps"`
	TargetKafka         int                                  `json:"targetKafka"`
	TargetClickHouse    int                                  `json:"targetClickHouse"`
	StartTime           time.Time                            `json:"startTime"`
	NodeData            map[string]*node_control.NodeMetrics `json:"nodeData"`
	ClickHouseMetrics   *clickhouse.ClickHouseMetrics        `json:"clickHouseMetrics,omitempty"`
	Mutex               sync.RWMutex
	Clients             map[*websocket.Conn]bool
	Broadcast           chan []byte
}

// Broadcast updates to all WebSocket clients

var AppState = &AppStates{
	IsSimulationRunning: false,
	CurrentProfile:      "medium",
	TargetEPS:           10000,
	TargetKafka:         5000,
	TargetClickHouse:    2000,
	NodeData:            make(map[string]*node_control.NodeMetrics),
	Clients:             make(map[*websocket.Conn]bool),
	Broadcast:           make(chan []byte, 256),
}

const (
	AppVersion = "1.0.0"
	StaticDir  = "./static"
	Port       = "164.52.213.158:8086"
)

var NodeManager = node_control.NewNodeManager()
var O11yManager = o11y_source_manager.NewO11ySourceManager()
var BinaryControl = bin_control.NewBinaryControl()

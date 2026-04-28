package generator

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"audit-service-logs-generator/internal/config"
	"audit-service-logs-generator/pkg/models"
)

type MessageGenerator struct {
	config         *config.Config
	sessionManager *SessionManager
}

func NewMessageGenerator(cfg *config.Config, sm *SessionManager) *MessageGenerator {
	return &MessageGenerator{
		config:         cfg,
		sessionManager: sm,
	}
}

func (mg *MessageGenerator) GenerateMessage(serviceId string, apiUrl string) ([]byte, error) {
	// Determine if this message should have null timestamps
	shouldNullTimestamps := rand.Intn(100) < mg.config.Execution.NullTimestampPercent
	
	var strtTimePtr *int64
	var endTimePtr *int64
	
	if !shouldNullTimestamps {
		startTime := time.Now().UnixMilli()
		// Calculate end time with random offset (10-100ms)
		randomOffset := int64(10 + rand.Intn(90))
		endTime := startTime + randomOffset
		strtTimePtr = &startTime
		endTimePtr = &endTime
	}
	// else: leave as nil for null timestamps

	msg := models.Message{
		MsgId:           mg.config.MessageTemplate.MsgId,
		TransId:         transId(),
		SessionTokenId:  mg.sessionManager.GetCurrentSession(),
		ServiceId:       serviceId,
		SrvcRqst:        mg.config.MessageTemplate.SrvcRqst,
		SrvcRspn:        mg.config.MessageTemplate.SrvcRspn,
		TrnsSts:         mg.config.ErrorTemplate.TrnsSts,
		ErrCd:           mg.config.ErrorTemplate.ErrCd,
		ErrMsg:          mg.config.ErrorTemplate.ErrMsg,
		CrtdBy:          mg.config.MessageTemplate.CrtdBy,
		CrtdOn:          time.Now().UnixMilli(),
		UsrId:           mg.config.MessageTemplate.UsrId,
		SystemName:      mg.config.ErrorTemplate.SystemName,
		ApiUrl:          apiUrl,
		RqstIdSent:      requestId(),
		RqstIdReceived:  requestId(),
		StrtTime:        strtTimePtr,
		EndTime:         endTimePtr,
	}

	// Marshal the inner message
	innerJSON, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// Wrap in a "message" field
	wrapper := map[string]string{
		"message": string(innerJSON),
	}

	return json.Marshal(wrapper)
}

func transId() string {
	return fmt.Sprintf("%d%s", time.Now().UnixNano()/1e6, randStr(8))
}

func requestId() string {
	return "SBIY" + uuid.New().String()[:16]
}

func randStr(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

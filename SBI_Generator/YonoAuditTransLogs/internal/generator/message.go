package generator

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"kafka-message-generator/internal/config"
	"kafka-message-generator/pkg/models"
)

// MessageType represents the type of message to generate
type MessageType int

const (
	ErrorOnly MessageType = iota
	TransOnly
	Both
)

// MessageGenerator generates YONO audit messages
type MessageGenerator struct {
	config           *config.Config
	sessionManager   *SessionManager
	templateSelector *TemplateSelector
}

// NewMessageGenerator creates a new message generator
func NewMessageGenerator(cfg *config.Config, sessionMgr *SessionManager) *MessageGenerator {
	return &MessageGenerator{
		config:           cfg,
		sessionManager:   sessionMgr,
		templateSelector: NewTemplateSelector(cfg),
	}
}

// GenerateMessage generates messages based on configuration
func (mg *MessageGenerator) GenerateMessage() ([]*models.MessageWrapper, error) {
	if mg.config.MessageType == "joint" {
		return mg.generateJointMessages()
	}

	if mg.config.MessageType == "access_log" {
		trnsID := GenerateTransactionID(mg.config.IDGeneration.TransactionIDPattern)
		sessnTknID, _ := mg.sessionManager.GetCurrentSession()
		raw := mg.generateAccessLogContent(
			trnsID,
			sessnTknID,
			GenerateRequestNo(mg.config.IDGeneration.RequestNoLength),
			GenerateTraceID("hex", 32),
		)

		wrapper := &models.MessageWrapper{
			Key:   trnsID,
			Topic: mg.config.Kafka.AccessLogTopic,
		}

		if mg.config.AccessLog.WrapInMessage {
			wrapper.Message = raw
		} else {
			wrapper.RawMessage = raw
		}
		return []*models.MessageWrapper{wrapper}, nil
	}

	msgType := mg.selectMessageType()
	var wrapper models.MessageWrapper

	// Generate shared fields
	trnsID := GenerateTransactionID(mg.config.IDGeneration.TransactionIDPattern)
	sessnTknID, cmndID := mg.sessionManager.GetCurrentSession()
	usrID := mg.templateSelector.SelectUserID()
	currentTime := time.Now().UnixMilli()

	shouldWrap := false

	switch msgType {
	case ErrorOnly:
		errorMsg := mg.generateErrorMessage(trnsID, sessnTknID, usrID, currentTime)
		errorJSON, err := errorMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		wrapper.YonoAdtError = &errorJSON
		shouldWrap = mg.config.Templates.Error.WrapInMessage

	case TransOnly:
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime,
			GenerateRequestNo(mg.config.IDGeneration.RequestNoLength),
			GenerateTraceID(mg.config.IDGeneration.TraceIDFormat, mg.config.IDGeneration.TraceIDLength),
			mg.templateSelector.SelectTransactionStatus(), cmndID)
		transJSON, err := transMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		wrapper.YonoAdtTrans = &transJSON
		shouldWrap = mg.config.Templates.Transaction.WrapInMessage

	case Both:
		errorMsg := mg.generateErrorMessage(trnsID, sessnTknID, usrID, currentTime)
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime,
			GenerateRequestNo(mg.config.IDGeneration.RequestNoLength),
			GenerateTraceID(mg.config.IDGeneration.TraceIDFormat, mg.config.IDGeneration.TraceIDLength),
			"error", cmndID)

		errorJSON, err := errorMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		transJSON, err := transMsg.ToJSONString()
		if err != nil {
			return nil, err
		}

		wrapper.YonoAdtError = &errorJSON
		wrapper.YonoAdtTrans = &transJSON
		shouldWrap = mg.config.Templates.Error.WrapInMessage || mg.config.Templates.Transaction.WrapInMessage
	}

	wrapper.Key = trnsID
	wrapper.Topic = mg.config.Kafka.Topic

	if shouldWrap {
		jsonBytes, _ := json.Marshal(wrapper)
		finalWrapper := &models.MessageWrapper{
			Message: string(jsonBytes),
			Key:     trnsID,
			Topic:   mg.config.Kafka.Topic,
		}
		return []*models.MessageWrapper{finalWrapper}, nil
	}

	return []*models.MessageWrapper{&wrapper}, nil
}



// generateJointMessages generates both audit/transaction and access log messages with shared IDs
func (mg *MessageGenerator) generateJointMessages() ([]*models.MessageWrapper, error) {
	// 1. Generate core shared identifiers
	trnsID := GenerateTransactionID(mg.config.IDGeneration.TransactionIDPattern)
	sessnTknID, cmndID := mg.sessionManager.GetCurrentSession()
	reqNo := GenerateRequestNo(mg.config.IDGeneration.RequestNoLength)
	traceID := GenerateTraceID("hex", 32)
	usrID := mg.templateSelector.SelectUserID()
	currentTime := time.Now().UnixMilli()

	// 2. Generate Audit/Transaction Message
	var auditWrapper models.MessageWrapper
	status := mg.templateSelector.SelectTransactionStatus()
	shouldWrapAudit := false

	if status == "success" {
		// Success implies TransOnly
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime, reqNo, traceID, "success", cmndID)
		transJSON, err := transMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		auditWrapper.YonoAdtTrans = &transJSON
		shouldWrapAudit = mg.config.Templates.Transaction.WrapInMessage
	} else {
		// Error implies Both (Trans + Error)
		errorMsg := mg.generateErrorMessage(trnsID, sessnTknID, usrID, currentTime)
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime, reqNo, traceID, "error", cmndID)

		errorJSON, err := errorMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		transJSON, err := transMsg.ToJSONString()
		if err != nil {
			return nil, err
		}

		auditWrapper.YonoAdtError = &errorJSON
		auditWrapper.YonoAdtTrans = &transJSON
		shouldWrapAudit = mg.config.Templates.Error.WrapInMessage || mg.config.Templates.Transaction.WrapInMessage
	}
	auditWrapper.Key = trnsID
	auditWrapper.Topic = mg.config.Kafka.Topic

	var finalAuditWrapper *models.MessageWrapper
	if shouldWrapAudit {
		jsonBytes, _ := json.Marshal(auditWrapper)
		finalAuditWrapper = &models.MessageWrapper{
			Message: string(jsonBytes),
			Key:     trnsID,
			Topic:   mg.config.Kafka.Topic,
		}
	} else {
		finalAuditWrapper = &auditWrapper
	}

	// 3. Generate Access Log Message
	accessRaw := mg.generateAccessLogContent(trnsID, sessnTknID, reqNo, traceID)
	accessWrapper := &models.MessageWrapper{
		Key:   trnsID,
		Topic: mg.config.Kafka.AccessLogTopic,
	}
	if mg.config.AccessLog.WrapInMessage {
		accessWrapper.Message = accessRaw
	} else {
		accessWrapper.RawMessage = accessRaw
	}

	// 4. Generate EIS Message
	eisMsg := mg.generateEISMessage(trnsID, sessnTknID, usrID, currentTime)
	eisJSON, err := eisMsg.ToJSONString()
	if err != nil {
		return nil, err
	}

	eisWrapper := &models.MessageWrapper{
		Key:   trnsID,
		Topic: mg.config.Kafka.EISTopic,
	}
	if mg.config.EIS.WrapInMessage {
		eisWrapper.Message = eisJSON
	} else {
		eisWrapper.RawMessage = eisJSON
	}

	return []*models.MessageWrapper{finalAuditWrapper, accessWrapper, eisWrapper}, nil
}

// selectMessageType selects message type based on distribution
func (mg *MessageGenerator) selectMessageType() MessageType {
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	random := int(n.Int64())

	if random < mg.config.Distribution.ErrorOnlyPercent {
		return ErrorOnly
	} else if random < mg.config.Distribution.ErrorOnlyPercent+mg.config.Distribution.TransOnlyPercent {
		return TransOnly
	}
	return Both
}

// generateErrorMessage generates an error message
func (mg *MessageGenerator) generateErrorMessage(trnsID, sessnTknID string, usrID, currentTime int64) *models.ErrorMessage {
	return &models.ErrorMessage{
		MsgID:      nil,
		TrnsID:     trnsID,
		SessnTknID: sessnTknID,
		ErrCD:      mg.templateSelector.SelectErrorCode(),
		UsrID:      usrID,
		ErrType:    mg.templateSelector.SelectErrorType(),
		ErrDscrptn: mg.templateSelector.SelectErrorDescription(),
		ErrDtls:    mg.templateSelector.SelectErrorDetails(),
		ErrTime:    currentTime,
		CrtdBy:     mg.config.Templates.Error.CreatedBy,
		Crtdon:     currentTime,
	}
}

// generateTransactionMessage generates a transaction message
func (mg *MessageGenerator) generateTransactionMessage(trnsID, sessnTknID string, usrID, currentTime int64, reqNo, traceID, status, cmndID string) *models.TransactionMessage {
	// Generate strtTime slightly before endTime (10-200ms)
	n, _ := rand.Int(rand.Reader, big.NewInt(191))
	offset := 10 + n.Int64()
	strtTime := currentTime - offset

	return &models.TransactionMessage{
		MsgID:          nil,
		TrnsID:         trnsID,
		SessnTknID:     sessnTknID,
		ClntIPAddress:  nil,
		ReqNo:          reqNo,
		UsrRltnshpNo:   mg.templateSelector.SelectUserRelationshipNumber(),
		TrnsStts:       status,
		StrtTime:       strtTime,
		EndTime:        currentTime,
		BizReqInput:    mg.templateSelector.SelectBizReqInput(),
		BizRegInput:    "",
		BizRespOutput:  mg.templateSelector.SelectBizRespOutput(),
		UsrID:          usrID,
		UsrTyp:         mg.templateSelector.SelectUserType(),
		CmndID:         cmndID,
		CrtdBy:         mg.config.Templates.Transaction.CreatedBy,
		Crtdon:         currentTime,
		ChnlID:         mg.templateSelector.SelectChannelID(),
		TraceID:        traceID,
	}
}

// generateAccessLogContent generates a JSON formatted access log message string
func (mg *MessageGenerator) generateAccessLogContent(trnsID, sessnTknID, reqNo, traceID string) string {
	loc := time.FixedZone("IST", 5*3600+30*60)
	currentTime := time.Now().In(loc)
	timestamp := currentTime.Format("02/Jan/2006 15:04:05") + fmt.Sprintf(":%03d", currentTime.Nanosecond()/1e6) + " IST"

	// Span ID remains random per message even in joint mode as it represents a specific hop
	spanID := GenerateTraceID("hex", 16)

	n, _ := rand.Int(rand.Reader, big.NewInt(191))
	execTime := 10 + n.Int64()

	podName := mg.templateSelector.SelectPodName()
	logPath := mg.templateSelector.SelectLogPath()
	logName := mg.templateSelector.SelectLogName()
	chnlID := mg.templateSelector.SelectAccessLogChannelID()
	apiURL := mg.templateSelector.SelectAPIUrl()
	status := mg.templateSelector.SelectHttpStatus()

	return fmt.Sprintf("%s trace_id=%s span_id=%s dt.trace_id= dt.span_id= dt.entity.process_group_instance= journey_id= pod_name=%s log_path=%s INFO %s: -, session-token-id: %s, request-no: %s, transaction-id: %s, channel-id: %d, channel-version: %s, api-url: %s, execution-time: %d, http-status: %s",
		timestamp, traceID, spanID, podName, logPath, logName, sessnTknID, reqNo, trnsID, chnlID, mg.config.AccessLog.ChannelVersion, apiURL, execTime, status)
}

// generateEISMessage generates an EIS message
func (mg *MessageGenerator) generateEISMessage(trnsID, sessnTknID string, usrID int64, currentTime int64) *models.EISMessage {
	rqstID := "SBIY" + GenerateRequestNo(20) // Approximate length from sample

	var strtTime, endTime interface{}
	// Randomly decide whether to include start/end times (approx 50% chance)
	n, _ := rand.Int(rand.Reader, big.NewInt(100))
	if n.Int64() < 50 {
		// Generate start/end time
		offset, _ := rand.Int(rand.Reader, big.NewInt(191)) // Similar to transaction logic
		latency := 10 + offset.Int64()
		strtTime = currentTime - latency
		endTime = currentTime
	}

	return &models.EISMessage{
		MsgID:          nil,
		TrnsID:         trnsID,
		SessnTknID:     sessnTknID,
		ServiceID:      mg.templateSelector.SelectEISServiceID(),
		SrvcRqst:       nil,
		SrvcRspn:       nil,
		TrnsSts:        "-1",
		ErrCD:          mg.templateSelector.SelectEISErrorCode(),
		ErrMsg:         mg.templateSelector.SelectEISErrorMessage(),
		CrtdBy:         mg.config.EIS.CreatedBy,
		Crtdon:         currentTime,
		UsrID:          usrID,
		SystemName:     mg.templateSelector.SelectEISSystemName(),
		APIUrl:         mg.templateSelector.SelectEISAPIUrl(),
		RqstIDSent:     rqstID,
		RqstIDReceived: rqstID,
		StrtTime:       strtTime,
		EndTime:        endTime,
	}
}

package generator

import (
	"crypto/rand"
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

// GenerateMessage generates a message based on distribution
func (mg *MessageGenerator) GenerateMessage() (*models.MessageWrapper, error) {
	msgType := mg.selectMessageType()

	var wrapper models.MessageWrapper

	// Generate shared fields
	trnsID := GenerateTransactionID(mg.config.IDGeneration.TransactionIDPattern)
	sessnTknID := mg.sessionManager.GetCurrentSession()
	usrID := mg.templateSelector.SelectUserID()
	currentTime := time.Now().UnixMilli()

	switch msgType {
	case ErrorOnly:
		errorMsg := mg.generateErrorMessage(trnsID, sessnTknID, usrID, currentTime)
		errorJSON, err := errorMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		wrapper.YonoAdtError = &errorJSON

	case TransOnly:
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime)
		transJSON, err := transMsg.ToJSONString()
		if err != nil {
			return nil, err
		}
		wrapper.YonoAdtTrans = &transJSON

	case Both:
		errorMsg := mg.generateErrorMessage(trnsID, sessnTknID, usrID, currentTime)
		transMsg := mg.generateTransactionMessage(trnsID, sessnTknID, usrID, currentTime)

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
	}

	return &wrapper, nil
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
func (mg *MessageGenerator) generateTransactionMessage(trnsID, sessnTknID string, usrID, currentTime int64) *models.TransactionMessage {
	// Generate strtTime slightly before endTime (10-200ms)
	n, _ := rand.Int(rand.Reader, big.NewInt(191))
	offset := 10 + n.Int64()
	strtTime := currentTime - offset

	return &models.TransactionMessage{
		MsgID:          nil,
		TrnsID:         trnsID,
		SessnTknID:     sessnTknID,
		ClntIPAddress:  nil,
		ReqNo:          GenerateRequestNo(mg.config.IDGeneration.RequestNoLength),
		UsrRltnshpNo:   mg.templateSelector.SelectUserRelationshipNumber(),
		TrnsStts:       mg.templateSelector.SelectTransactionStatus(),
		StrtTime:       strtTime,
		EndTime:        currentTime,
		BizReqInput:    mg.templateSelector.SelectBizReqInput(),
		BizRegInput:    "",
		BizRespOutput:  mg.templateSelector.SelectBizRespOutput(),
		UsrID:          usrID,
		UsrTyp:         mg.templateSelector.SelectUserType(),
		CmndID:         mg.templateSelector.SelectCommandID(),
		CrtdBy:         mg.config.Templates.Transaction.CreatedBy,
		Crtdon:         currentTime,
		ChnlID:         mg.templateSelector.SelectChannelID(),
		TraceID:        GenerateTraceID(mg.config.IDGeneration.TraceIDFormat, mg.config.IDGeneration.TraceIDLength),
	}
}

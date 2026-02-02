package models

import "encoding/json"

// TransactionMessage represents the yono_adt_trans schema
type TransactionMessage struct {
	MsgID          interface{} `json:"msgId"`          // Always null
	TrnsID         string      `json:"trnsId"`         // Transaction ID
	SessnTknID     string      `json:"sessnTknId"`     // Session token ID
	ClntIPAddress  interface{} `json:"clntIpAddress"`  // Always null
	ReqNo          string      `json:"reqNo"`          // Request number
	UsrRltnshpNo   string      `json:"usrRltnshpNo"`   // User relationship number
	TrnsStts       string      `json:"trnsStts"`       // Transaction status
	StrtTime       int64       `json:"strtTime"`       // Start time (epoch millis)
	EndTime        int64       `json:"endTime"`        // End time (epoch millis)
	BizReqInput    string      `json:"bizReqInput"`    // Business request input (Base64)
	BizRegInput    string      `json:"bizRegInput"`    // Business reg input (Base64, optional)
	BizRespOutput  string      `json:"bizRespOutput"`  // Business response output (Base64)
	UsrID          int64       `json:"usrId"`          // User ID
	UsrTyp         string      `json:"usrTyp"`         // User type
	CmndID         string      `json:"cmndId"`         // Command ID
	CrtdBy         string      `json:"crtdBy"`         // Created by
	Crtdon         int64       `json:"crtdon"`         // Created on (epoch millis)
	ChnlID         int         `json:"chnlId"`         // Channel ID
	TraceID        string      `json:"traceId"`        // Trace ID
}

// ToJSONString converts the transaction message to a JSON string
func (t *TransactionMessage) ToJSONString() (string, error) {
	bytes, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

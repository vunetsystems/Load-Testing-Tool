package models

import "encoding/json"

// EISMessage represents the EIS message schema
type EISMessage struct {
	MsgID          interface{} `json:"msgId"`          // Always null
	TrnsID         string      `json:"transId"`        // Transaction ID (Note field name difference vs TransactionMessage)
	SessnTknID     string      `json:"sessnTknId"`     // Session token ID
	ServiceID      string      `json:"serviceId"`      // Service ID
	SrvcRqst       interface{} `json:"srvcRqst"`       // Service Request (null)
	SrvcRspn       interface{} `json:"srvcRspn"`       // Service Response (null)
	TrnsSts        string      `json:"trnsSts"`        // Transaction Status
	ErrCD          string      `json:"errCd"`          // Error Code
	ErrMsg         string      `json:"errMsg"`         // Error Message
	CrtdBy         string      `json:"crtdBy"`         // Created By
	Crtdon         int64       `json:"crtdOn"`         // Created On (epoch millis)
	UsrID          int64       `json:"usrId"`          // User ID
	SystemName     string      `json:"systemName"`     // System Name
	APIUrl         string      `json:"apiUrl"`         // API URL
	RqstIDSent     string      `json:"rqstIdSent"`     // Request ID Sent
	RqstIDReceived string      `json:"rqstIdReceived"` // Request ID Received
	StrtTime       interface{} `json:"strtTime"`       // Start Time (null)
	EndTime        interface{} `json:"endTime"`        // End Time (null)
}

// ToJSONString converts the EIS message to a JSON string
func (e *EISMessage) ToJSONString() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

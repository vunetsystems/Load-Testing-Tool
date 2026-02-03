package models

import "encoding/json"

// ErrorMessage represents the yono_adt_error schema
type ErrorMessage struct {
	MsgID        interface{} `json:"msgId"`        // Always null
	TrnsID       string      `json:"trnsId"`       // Transaction ID
	SessnTknID   string      `json:"sessnTknId"`   // Session token ID
	ErrCD        string      `json:"errCd"`        // Error code
	UsrID        int64       `json:"usrId"`        // User ID
	ErrType      string      `json:"errType"`      // Error type
	ErrDscrptn   string      `json:"errDscrptn"`   // Error description (stringified JSON)
	ErrDtls      string      `json:"errDtls"`      // Error details
	ErrTime      int64       `json:"errTime"`      // Error timestamp (epoch millis)
	CrtdBy       string      `json:"crtdBy"`       // Created by
	Crtdon       int64       `json:"crtdOn"`       // Created on (epoch millis)
}

// ToJSONString converts the error message to a JSON string
func (e *ErrorMessage) ToJSONString() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

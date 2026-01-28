package models

import "encoding/json"

// MessageWrapper represents the top-level Kafka message
type MessageWrapper struct {
	YonoAdtError *string `json:"yono_adt_error,omitempty"` // Optional error message (JSON string)
	YonoAdtTrans *string `json:"yono_adt_trans,omitempty"` // Optional transaction message (JSON string)
}

// ToJSON converts the wrapper to JSON bytes
func (w *MessageWrapper) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

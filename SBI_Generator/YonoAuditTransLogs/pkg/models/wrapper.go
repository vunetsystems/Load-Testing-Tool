package models

import "encoding/json"

// MessageWrapper represents the top-level Kafka message
type MessageWrapper struct {
	YonoAdtError *string `json:"yono_adt_error,omitempty"` // Optional error message (JSON string)
	YonoAdtTrans *string `json:"yono_adt_trans,omitempty"` // Optional transaction message (JSON string)
	YonoAdtEis   *string `json:"yono_adt_eis,omitempty"`   // Optional EIS message (JSON string)
	Message      string  `json:"message,omitempty"`        // For JSON formatted plain text messages
	RawMessage   string  `json:"-"`                      // For plain text messages
	Key          string  `json:"-"`                      // Optional Kafka key
	Topic        string  `json:"-"`                      // Destination Kafka topic
}

// ToJSON converts the wrapper to JSON bytes or returns raw message
func (w *MessageWrapper) ToJSON() ([]byte, error) {
	if w.RawMessage != "" {
		return []byte(w.RawMessage), nil
	}
	return json.Marshal(w)
}

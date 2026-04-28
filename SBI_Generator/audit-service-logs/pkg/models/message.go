package models

// Message represents the audit log message structure
type Message struct {
	MsgId           interface{} `json:"msgId"`
	TransId         string      `json:"transId"`
	SessionTokenId  string      `json:"sessnTknId"`
	ServiceId       string      `json:"serviceId"`
	SrvcRqst        interface{} `json:"srvcRqst"`
	SrvcRspn        interface{} `json:"srvcRspn"`
	TrnsSts         string      `json:"trnsSts"`
	ErrCd           string      `json:"errCd"`
	ErrMsg          string      `json:"errMsg"`
	CrtdBy          string      `json:"crtdBy"`
	CrtdOn          int64       `json:"crtdOn"`
	UsrId           interface{} `json:"usrId"`
	SystemName      string      `json:"systemName"`
	ApiUrl          string      `json:"apiUrl"`
	RqstIdSent      string      `json:"rqstIdSent"`
	RqstIdReceived  string      `json:"rqstIdReceived"`
	StrtTime        *int64      `json:"strtTime"`
	EndTime         *int64      `json:"endTime"`
}

package assistant

type ClientReq struct {
	ReqType string `json:"reqType"`
	Payload string `json:"payload"`
}

type AssistantReq struct {
	ReqType     string `json:"rewType"`
	Name        string `json:"name"`
	Instruction string `json:"instruction"`
}

type ClientRes struct {
	ReqType string `json:"reqType"`
	Payload string `json:"payload"`
}

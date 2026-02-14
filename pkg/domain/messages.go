package domain

type RFIDMessage struct {
	ID   string `json:"id"`
	Data string `json:"data,omitempty"`
	Type string `json:"type"`
}

package pubsub

// BaseMessage is something
type BaseMessage struct {
	Type string `json:"type"`
}

// ListenData is something
type ListenData struct {
	Topics    []string `json:"topics"`
	AuthToken string   `json:"auth_token,omitempty"`
}

// RequestMessage is something
type RequestMessage struct {
	BaseMessage
	Nonce string     `json:"nonce,omitempty"`
	Data  ListenData `json:"data"`
}

// ResponseMessage is something
type ResponseMessage struct {
	BaseMessage
	Nonce string `json:"nonce,omitempty"`
	Error string `json:"error"`
}

// MessageData is something
type MessageData struct {
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

// MessageMessage is something
type MessageMessage struct {
	BaseMessage
	Data MessageData `json:"data"`
}

package main

import "fmt"

// Topic is something
type Topic struct {
	Name      string
	Nonce     string
	AuthToken string
	Callback  TopicCallback
}

// TopicCallback is something
type TopicCallback func(MessageData)

// ListenMessage is something
func (t *Topic) ListenMessage() RequestMessage {
	return RequestMessage{
		BaseMessage: BaseMessage{
			Type: "LISTEN",
		},
		Nonce: t.Nonce,
		Data: ListenData{
			Topics:    []string{t.Name},
			AuthToken: t.AuthToken,
		},
	}
}

// UnlistenMessage is something
func (t *Topic) UnlistenMessage() RequestMessage {
	return RequestMessage{
		BaseMessage: BaseMessage{
			Type: "UNLISTEN",
		},
		Nonce: t.Nonce,
		Data: ListenData{
			Topics: []string{t.Name},
		},
	}
}

// Identifier is something
func (t *Topic) Identifier() string {
	return mustHashString(fmt.Sprintf("%s:%s", t.Name, t.AuthToken))
}

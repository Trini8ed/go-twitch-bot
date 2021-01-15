package pubsub

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Twitch Helix API variables
const (
	maxTopics       = 50
	nonceLength     = 16
	pingInterval    = time.Minute * 4
	pongDeadline    = time.Second * 10
	twitchPubSubURL = "wss://pubsub-edge.twitch.tv"
)

// Custom error messages for API
var (
	// ErrPingTimeout is when ping timed out.
	// OnError info: time.Duration of ping timeout
	ErrPingTimeout = errors.New("PING timed out")

	// is when an event occurred without a corresponding topic registered.
	ErrInvalidTopic = errors.New("topic not found")

	// ErrTooManyTopics is when a client attempted to listen to too many topics.
	ErrTooManyTopics = errors.New("too many topics")

	// ErrDuplicateTopic is when a client attempted to listen to a duplicate topic.
	ErrDuplicateTopic = errors.New("duplicate topic")

	// ErrBadMessage PubSub ERR_BADMESSAGE response.
	// OnError info: Topic that triggered the error.
	ErrBadMessage = errors.New("pubsub ERR_BADMESSAGE")

	// ErrBadAuth PubSub ERR_BADAUTH response.
	// OnError info: Topic that triggered the error.
	ErrBadAuth = errors.New("pubsub ERR_BADAUTH")

	// ErrServer PubSub ERR_SERVER response.
	// OnError info: Topic that triggered the error.
	ErrServer = errors.New("pubsub ERR_SERVER")

	// ErrBadTopic PubSub ERR_BADTOPIC response.
	// OnError info: Topic that triggered the error.
	ErrBadTopic = errors.New("pubsub ERR_BADTOPIC")
)

// Sub is something
type Sub interface {
	Listen(topic string, callback TopicCallback) (*Topic, error)
	ListenMany(callback TopicCallback, topics ...string) ([]*Topic, error)

	Unlisten(topic string) error
	UnlistenMany(topics ...string) error

	IsListening(topic string) bool
}

// Conn is something
type Conn struct {
	ws        *BasicWebsocket
	authToken string

	pingDone chan bool
	pongChan chan bool

	topics      []*Topic
	topicsMutex sync.RWMutex

	// Called on connection connect
	OnConnect func()
	// Called on error
	OnError func(err error, info interface{})
}

// NewConn is something
func NewConn(authToken string, header http.Header) *Conn {
	ws := NewBasicWebsocket(twitchPubSubURL, header)
	ws.AutoReconnect = true

	conn := &Conn{
		ws:        ws,
		authToken: authToken,

		pingDone: make(chan bool),
		pongChan: make(chan bool),

		topics: make([]*Topic, 0),

		OnConnect: func() {},
		OnError:   func(err error, info interface{}) {},
	}

	ws.OnConnect = conn.connectHandler
	ws.OnMessage = conn.rawMessageHandler
	ws.OnError = func(err error) {
		conn.OnError(err, nil)
	}

	return conn
}

func (c *Conn) connectHandler() {
	// stop any current ping goroutines
	select {
	case c.pingDone <- true:
	default:
		break
	}

	c.pingDone = c.startPing()

	t, err := c.listenToAllTopics()
	if err != nil {
		c.OnError(err, t)
	}

	c.OnConnect()
}

func (c *Conn) listenToTopic(topic *Topic) error {
	return c.ws.SendJSON(topic.ListenMessage())
}

func (c *Conn) unlistenToTopic(topic *Topic) error {
	return c.ws.SendJSON(topic.UnlistenMessage())
}

func (c *Conn) listenToAllTopics() (*Topic, error) {
	c.topicsMutex.RLock()
	defer c.topicsMutex.RUnlock()
	for _, topic := range c.topics {
		err := c.listenToTopic(topic)
		if err != nil {
			return topic, err
		}
	}
	return nil, nil
}

func (c *Conn) rawMessageHandler(data []byte) (err error) {
	base := BaseMessage{}
	err = json.Unmarshal(data, &base)
	if err != nil {
		return
	}

	switch base.Type {
	case "RECONNECT":
		return c.ws.Reconnect()
	case "RESPONSE":
		return c.onResponse(data)
	case "MESSAGE":
		return c.onMessage(data)
	case "PONG":
		c.onPong()
		return
	default:
		return
	}
}

func (c *Conn) onPong() {
	c.pongChan <- true
}

func (c *Conn) sendPing() error {
	message := &BaseMessage{
		Type: "PING",
	}

	err := c.ws.SendJSON(message)
	if err != nil {
		return err
	}

	go func() {
		timer := time.NewTimer(pongDeadline)
		defer timer.Stop()
		for {
			select {
			case <-c.pongChan:
				return
			case <-timer.C:
				if !c.ws.IsConnected() {
					return
				}

				c.OnError(ErrPingTimeout, pongDeadline)
				_ = c.ws.Reconnect()
				return
			}
		}
	}()

	return nil
}

func (c *Conn) startPing() chan bool {
	doneChan := make(chan bool, 1)
	go func() {
		fire := func() {
			// Sleep 0-3 seconds for jitter
			time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)
			_ = c.sendPing()
		}

		ticker := time.NewTicker(pingInterval)
		fire()
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fire()
			case <-doneChan:
				return
			}
		}
	}()

	return doneChan
}

func (c *Conn) onResponse(data []byte) error {
	response := ResponseMessage{}
	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	if response.Error != "" {
		errorTopic := c.getTopicByNonce(response.Nonce)
		if errorTopic == nil || !c.removeTopic(errorTopic) {
			return fmt.Errorf("received error for invalid nonce %q: %w", response.Nonce, ErrInvalidTopic)
		}

		switch response.Error {
		case "ERR_BADMESSAGE":
			c.OnError(ErrBadMessage, errorTopic)
		case "ERR_BADAUTH":
			c.OnError(ErrBadAuth, errorTopic)
		case "ERR_SERVER":
			c.OnError(ErrServer, errorTopic)
		case "ERR_BADTOPIC":
			c.OnError(ErrBadTopic, errorTopic)
		}
	}

	return nil
}

func (c *Conn) onMessage(data []byte) error {
	message := MessageMessage{}
	err := json.Unmarshal(data, &message)
	if err != nil {
		return err
	}

	topic := c.getTopicByName(message.Data.Topic)
	if topic == nil {
		return fmt.Errorf("received message for invalid topic %q: %w", message.Data.Topic, ErrInvalidTopic)
	}

	go topic.Callback(message.Data)
	return nil
}

func (c *Conn) getTopicByNonce(nonce string) *Topic {
	c.topicsMutex.RLock()
	defer c.topicsMutex.RUnlock()

	for _, topic := range c.topics {
		if topic.Nonce == nonce {
			return topic
		}
	}
	return nil
}

func (c *Conn) getTopicByName(name string) *Topic {
	c.topicsMutex.RLock()
	defer c.topicsMutex.RUnlock()

	for _, topic := range c.topics {
		if topic.Name == name {
			return topic
		}
	}
	return nil
}

// Listen is something
func (c *Conn) Listen(topic string, callback TopicCallback) (*Topic, error) {
	if c.Capacity() == 0 {
		return nil, ErrTooManyTopics
	}

	if c.getTopicByName(topic) != nil {
		return nil, fmt.Errorf("listen topic %q: %w", topic, ErrDuplicateTopic)
	}

	nonce, err := GenerateRandomNonce(nonceLength)
	if err != nil {
		return nil, err
	}

	newTopic := &Topic{
		Name:      topic,
		Nonce:     nonce,
		AuthToken: c.authToken,
		Callback:  callback,
	}

	c.topicsMutex.Lock()
	c.topics = append(c.topics, newTopic)
	c.topicsMutex.Unlock()

	if c.ws.IsConnected() {
		err = c.listenToTopic(newTopic)
		if err != nil {
			return nil, err
		}
	}
	return newTopic, nil
}

// ListenMany is something
func (c *Conn) ListenMany(callback TopicCallback, topics ...string) ([]*Topic, error) {
	var returnedTopics []*Topic
	for _, topic := range topics {
		t, err := c.Listen(topic, callback)
		if err != nil {
			return nil, err
		}
		returnedTopics = append(returnedTopics, t)
	}
	return returnedTopics, nil
}

func (c *Conn) removeTopic(topic *Topic) bool {
	c.topicsMutex.Lock()
	defer c.topicsMutex.Unlock()

	index := -1
	for i, t := range c.topics {
		if t.Identifier() == topic.Identifier() {
			index = i
			break
		}
	}

	if index == -1 {
		return false
	}

	// remove item at index
	c.topics[index] = c.topics[len(c.topics)-1]
	c.topics[len(c.topics)-1] = nil
	c.topics = c.topics[:len(c.topics)-1]

	return true
}

// Unlisten is something
func (c *Conn) Unlisten(topic string) error {
	if c.getTopicByName(topic) == nil {
		return fmt.Errorf("unlisten topic %q: %w", topic, ErrInvalidTopic)
	}

	nonce, err := GenerateRandomNonce(nonceLength)
	if err != nil {
		return err
	}

	matchTopic := &Topic{
		Name:      topic,
		Nonce:     nonce,
		AuthToken: c.authToken,
	}

	c.removeTopic(matchTopic)

	if c.ws.IsConnected() {
		err = c.unlistenToTopic(matchTopic)
		if err != nil {
			return err
		}
	}
	return nil
}

// UnlistenMany is something
func (c *Conn) UnlistenMany(topics ...string) error {
	for _, topic := range topics {
		err := c.Unlisten(topic)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsListening is something
func (c *Conn) IsListening(topic string) bool {
	return c.getTopicByName(topic) != nil
}

// Count returns the topic count
func (c *Conn) Count() int {
	c.topicsMutex.RLock()
	defer c.topicsMutex.RUnlock()
	return len(c.topics)
}

// Capacity returns the capacity for more topics
func (c *Conn) Capacity() int {
	c.topicsMutex.RLock()
	defer c.topicsMutex.RUnlock()
	return maxTopics - len(c.topics)
}

// Start is something
func (c *Conn) Start() (err error) {
	err = c.ws.Connect()
	if err != nil {
		return
	}

	return
}

// Stop is something
func (c *Conn) Stop() {
	if !c.ws.IsConnected() {
		return
	}

	c.ws.ForceDisconnect()
	c.pingDone <- true
}

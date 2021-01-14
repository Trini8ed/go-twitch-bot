package pubsub

import (
	"fmt"
	"net/http"
	"sync"
)

// PubSubPool is something
type PubSubPool struct {
	running      bool
	runningMutex sync.Mutex

	connections      []*PubSubConn
	connectionsMutex sync.RWMutex

	authToken string
	header    http.Header

	// Called on pool start
	OnStart func()
	// Called on individual connection connect/reconnect
	OnConnect func(conn *PubSubConn)
	// Called on errors
	OnError func(*PubSubConn, error, interface{})
}

// NewPubSubPool is something
func NewPubSubPool(authToken string, header http.Header) *PubSubPool {
	return &PubSubPool{
		running:     false,
		connections: make([]*PubSubConn, 0),
		authToken:   authToken,
		header:      header,

		OnStart:   func() {},
		OnConnect: func(conn *PubSubConn) {},
		OnError:   func(conn *PubSubConn, err error, info interface{}) {},
	}
}

func (p *PubSubPool) getTopicByName(name string) (*Topic, *PubSubConn) {
	p.connectionsMutex.RLock()
	defer p.connectionsMutex.RUnlock()
	for _, conn := range p.connections {
		topic := conn.getTopicByName(name)
		if topic != nil {
			return topic, conn
		}
	}
	return nil, nil
}

func (p *PubSubPool) createNewConnection() *PubSubConn {
	p.connectionsMutex.Lock()
	defer p.connectionsMutex.Unlock()

	// create and configure connection
	newConn := NewPubSubConn(p.authToken, p.header)
	newConn.OnConnect = func() {
		p.OnConnect(newConn)
	}
	newConn.OnError = func(err error, info interface{}) {
		p.OnError(newConn, err, info)
	}
	p.connections = append(p.connections, newConn)

	// start the new connection if already running
	p.runningMutex.Lock()
	if p.running {
		err := newConn.Start()
		if err != nil {
			p.OnError(newConn, err, nil)
		}
	}
	p.runningMutex.Unlock()

	return newConn
}

func (p *PubSubPool) getTargetConnection() (targetConnection *PubSubConn) {
	p.connectionsMutex.RLock()
	if len(p.connections) == 0 {
		// No connections yet
		p.connectionsMutex.RUnlock()
		targetConnection = p.createNewConnection()
	} else {
		// find first connection with available space
		for _, conn := range p.connections {
			if conn.Capacity() > 0 {
				targetConnection = conn
				p.connectionsMutex.RUnlock()
				return
			}
		}

		// must create new connection now
		p.connectionsMutex.RUnlock()
		targetConnection = p.createNewConnection()
	}
	return
}

// Listen is something
func (p *PubSubPool) Listen(topic string, callback TopicCallback) (*Topic, error) {
	if t, _ := p.getTopicByName(topic); t != nil {
		return nil, fmt.Errorf("listen topic %s: %w", topic, ErrDuplicateTopic)
	}

	targetConnection := p.getTargetConnection()
	t, err := targetConnection.Listen(topic, callback)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListenMany is something
func (p *PubSubPool) ListenMany(callback TopicCallback, topics ...string) ([]*Topic, error) {
	var returnedTopics []*Topic
	for _, topic := range topics {
		t, err := p.Listen(topic, callback)
		if err != nil {
			return nil, err
		}
		returnedTopics = append(returnedTopics, t)
	}
	return returnedTopics, nil
}

// Unlisten is something
func (p *PubSubPool) Unlisten(topic string) error {
	t, conn := p.getTopicByName(topic)
	if t == nil {
		return fmt.Errorf("unlisten topic %s: %w", topic, ErrInvalidTopic)
	}

	err := conn.Unlisten(topic)
	if err != nil {
		return err
	}
	return nil
}

// UnlistenMany is something
func (p *PubSubPool) UnlistenMany(topics ...string) error {
	for _, topic := range topics {
		err := p.Unlisten(topic)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsListening is something
func (p *PubSubPool) IsListening(topic string) bool {
	t, _ := p.getTopicByName(topic)
	return t != nil
}

// Start is something
func (p *PubSubPool) Start() (err error) {
	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if p.running {
		return
	}

	p.connectionsMutex.RLock()
	defer func() {
		p.connectionsMutex.RUnlock()
		if err == nil {
			p.OnStart()
		}
	}()

	for _, conn := range p.connections {
		err = conn.Start()
		if err != nil {
			return
		}
	}

	p.running = true
	return
}

// Stop is something
func (p *PubSubPool) Stop() {
	p.runningMutex.Lock()
	defer p.runningMutex.Unlock()

	if !p.running {
		return
	}

	p.connectionsMutex.RLock()
	defer p.connectionsMutex.RUnlock()
	for _, conn := range p.connections {
		conn.Stop()
	}

	p.running = false
	return
}

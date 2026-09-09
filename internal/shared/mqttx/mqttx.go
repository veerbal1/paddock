// Package mqttx is the post office window: one shared way for both
// binaries to talk to the MQTT broker. collarsim publishes, backend
// subscribes. Neither touches Paho directly.
package mqttx

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/veerbal1/paddock/internal/shared/telemetry"
)

// PingTopic is the address on the envelope: which farm, which collar.
func PingTopic(farmID, cowID string) string {
	return fmt.Sprintf("farm/%s/collar/%s/ping", farmID, cowID)
}

// PingPattern matches every collar on one farm: farm/f1/collar/+/ping.
// The + is a single-level wildcard — any cow ID fits.
func PingPattern(farmID string) string {
	return fmt.Sprintf("farm/%s/collar/+/ping", farmID)
}

// FenceTopic is the notice board: which farm, which paddock.
func FenceTopic(farmID, paddockID string) string {
	return fmt.Sprintf("farm/%s/fence/%s", farmID, paddockID)
}

// StatusTopic is where a collar's liveness is posted: one topic per collar.
// Two writers share it — the collar itself when it connects, and the broker
// on the collar's behalf when the line drops without a goodbye.
func StatusTopic(farmID, collarID string) string {
	return fmt.Sprintf("farm/%s/collar/%s/status", farmID, collarID)
}

// StatusPattern matches every collar's status on one farm.
func StatusPattern(farmID string) string {
	return fmt.Sprintf("farm/%s/collar/+/status", farmID)
}

// The only two things a status topic ever says.
const (
	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
)

// FenceMsg is what rides on the board: version + drawing.
type FenceMsg struct {
	V       int             `json:"v"`
	Version int             `json:"version"`
	Polygon json.RawMessage `json:"polygon"`
}

// PublishFence pins the drawing on the notice board (retained=true).
// Late joiners read it instantly; nobody needs catch-up code.
func (m *Client) PublishFence(farmID, paddockID string, version int, polygon []byte) error {
	raw, err := json.Marshal(FenceMsg{V: 1, Version: version, Polygon: polygon})
	if err != nil {
		return err
	}
	tok := m.c.Publish(FenceTopic(farmID, paddockID), 1, true, raw)
	tok.Wait()
	return tok.Error()
}

// ClearFence tears the notice down (empty retained). Without this the
// ghost fence haunts every reconnect after a paddock delete.
func (m *Client) ClearFence(farmID, paddockID string) error {
	tok := m.c.Publish(FenceTopic(farmID, paddockID), 1, true, []byte{})
	tok.Wait()
	return tok.Error()
}

// Handler is what the backend passes in: "when a ping arrives, call me."
// Giving your function is not importing — the arrow points from your
// code to Paho, never back.
type Handler func(telemetry.Ping)

// farmFromTopic pulls "f1" out of "farm/f1/collar/c7/ping".
// The topic is the broker-seen truth; the payload is the collar's claim.
// They must match — mismatch means a liar, and Loop 4 drops it.
func farmFromTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 || parts[0] != "farm" || parts[2] != "collar" || parts[4] != "ping" {
		return "", false
	}
	return parts[1], true
}

// collarFromStatusTopic pulls "c7" out of "farm/f1/collar/c7/status".
func collarFromStatusTopic(topic string) (string, bool) {
	parts := strings.Split(topic, "/")
	if len(parts) != 5 || parts[0] != "farm" || parts[2] != "collar" || parts[4] != "status" {
		return "", false
	}
	return parts[3], true
}

// StatusHandler is called with one collar's id and its new status.
type StatusHandler func(collarID, status string)

// SubscribeStatus watches every collar's liveness on one farm. QoS 1 and
// retained on the publish side, so a late-joining backend is told the
// current state of the fleet the moment it subscribes.
func (m *Client) SubscribeStatus(farmID string, h StatusHandler) error {
	tok := m.c.Subscribe(StatusPattern(farmID), 1, func(_ paho.Client, msg paho.Message) {
		collarID, ok := collarFromStatusTopic(msg.Topic())
		if !ok {
			return
		}
		h(collarID, string(msg.Payload()))
	})
	tok.Wait()
	return tok.Error()
}

// Subscribe tells the broker "send this pattern to me" and registers
// the handler Paho calls on every arrival. QoS 0: same as publish side.
// Every ping leaves here stamped with the farm the topic said.
func (m *Client) Subscribe(farmID string, h Handler) error {
	tok := m.c.Subscribe(PingPattern(farmID), 0, func(_ paho.Client, msg paho.Message) {
		var p telemetry.Ping
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			return
		}
		farm, ok := farmFromTopic(msg.Topic())
		if !ok {
			return
		}
		p.FarmID = farm
		p.V = 1
		h(p)
	})
	tok.Wait()
	return tok.Error()
}

// Client holds the open line to the broker. Connect once at startup,
// then PublishPing every tick.
//
// farmID and collarID are set only on a collar's client (see NewCollar).
// They are what Close needs in order to say goodbye properly.
type Client struct {
	c        paho.Client
	farmID   string
	collarID string
}

// options are the settings every client shares. clientID must be unique
// across the whole broker: two clients connecting with the same id get each
// other kicked off in a loop that never settles.
func options(broker, clientID string) *paho.ClientOptions {
	opts := paho.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	return opts
}

// New dials the broker and waits for the handshake. broker is the
// address of the post office, e.g. "tcp://localhost:1883".
func New(broker, clientID string) (*Client, error) {
	c := paho.NewClient(options(broker, clientID))
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, tok.Error()
	}
	return &Client{c: c}, nil
}

// NewCollar dials the broker as one collar, with its own line, its own id,
// and its own will.
//
// The will is the whole point: the broker holds it, and publishes it on this
// collar's behalf if the line drops without a goodbye — a flat battery, a
// hill, a dead radio. A collar cannot report its own death, so the broker
// reports it instead. Retained, so a backend that starts later still learns
// the collar is dark.
func NewCollar(broker, farmID, collarID string) (*Client, error) {
	opts := options(broker, "collar-"+collarID)
	opts.SetWill(StatusTopic(farmID, collarID), StatusDisconnected, 1, true)

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, tok.Error()
	}
	m := &Client{c: c, farmID: farmID, collarID: collarID}

	// The collar writes its own life; the broker writes its death. Without
	// this the topic would only ever say "disconnected".
	if err := m.publishStatus(StatusConnected); err != nil {
		m.c.Disconnect(0)
		return nil, err
	}
	return m, nil
}

// publishStatus pins this collar's liveness on its own topic.
func (m *Client) publishStatus(status string) error {
	tok := m.c.Publish(StatusTopic(m.farmID, m.collarID), 1, true, status)
	tok.Wait()
	return tok.Error()
}

// PublishPing wraps one ping as JSON and drops it at the broker.
// QoS 0: fire and forget — losing one position is fine, next second
// brings a fresh one.
func (m *Client) PublishPing(farmID string, p telemetry.Ping) error {
	raw, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tok := m.c.Publish(PingTopic(farmID, p.CowID), 0, false, raw)
	tok.Wait()
	return tok.Error()
}

// Close hangs up the line. A collar says goodbye on the way out: a clean
// shutdown does not fire the will, so without this the topic would say
// "connected" forever after a deliberate stop — the same ghost problem a
// retained fence has after a paddock is deleted.
func (m *Client) Close() {
	if m.collarID != "" {
		if err := m.publishStatus(StatusDisconnected); err != nil {
			log.Printf("mqttx: collar %s goodbye failed: %v", m.collarID, err)
		}
	}
	m.c.Disconnect(250)
}

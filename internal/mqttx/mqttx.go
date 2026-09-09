// Package mqttx is the post office window: one shared way for both
// binaries to talk to the MQTT broker. collarsim publishes, backend
// subscribes. Neither touches Paho directly.
package mqttx

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/veerbal1/paddock/internal/telemetry"
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
type Client struct {
	c paho.Client
}

// New dials the broker and waits for the handshake. broker is the
// address of the post office, e.g. "tcp://localhost:1883".
func New(broker, clientID string) (*Client, error) {
	opts := paho.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)

	c := paho.NewClient(opts)
	if tok := c.Connect(); tok.Wait() && tok.Error() != nil {
		return nil, tok.Error()
	}
	return &Client{c: c}, nil
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

// Close hangs up the line.
func (m *Client) Close() {
	m.c.Disconnect(250)
}

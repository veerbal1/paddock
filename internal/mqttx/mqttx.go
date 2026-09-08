// Package mqttx is the post office window: one shared way for both
// binaries to talk to the MQTT broker. collarsim publishes, backend
// subscribes. Neither touches Paho directly.
package mqttx

import (
	"encoding/json"
	"fmt"
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

// Handler is what the backend passes in: "when a ping arrives, call me."
// Giving your function is not importing — the arrow points from your
// code to Paho, never back.
type Handler func(telemetry.Ping)

// Subscribe tells the broker "send this pattern to me" and registers
// the handler Paho calls on every arrival. QoS 0: same as publish side.
func (m *Client) Subscribe(farmID string, h Handler) error {
	tok := m.c.Subscribe(PingPattern(farmID), 0, func(_ paho.Client, msg paho.Message) {
		var p telemetry.Ping
		if err := json.Unmarshal(msg.Payload(), &p); err != nil {
			return
		}
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

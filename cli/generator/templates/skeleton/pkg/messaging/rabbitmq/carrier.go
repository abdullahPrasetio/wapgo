//go:build ignore

package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

// amqpTableCarrier adapts amqp.Table to propagation.TextMapCarrier so OTel
// trace context can be injected into / extracted from AMQP message headers.
type amqpTableCarrier amqp.Table

func (c amqpTableCarrier) Get(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (c amqpTableCarrier) Set(key, val string) { c[key] = val }

func (c amqpTableCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

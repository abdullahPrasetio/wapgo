package kafka

import kafkago "github.com/segmentio/kafka-go"

// kafkaHeaderCarrier adapts a *[]kafkago.Header to propagation.TextMapCarrier
// so OTel trace context can be injected into Kafka message headers.
type kafkaHeaderCarrier struct {
	headers *[]kafkago.Header
}

func (c *kafkaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaHeaderCarrier) Set(key, val string) {
	for i, h := range *c.headers {
		if h.Key == key {
			(*c.headers)[i].Value = []byte(val)
			return
		}
	}
	*c.headers = append(*c.headers, kafkago.Header{Key: key, Value: []byte(val)})
}

func (c *kafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(*c.headers))
	for i, h := range *c.headers {
		keys[i] = h.Key
	}
	return keys
}

// ExtractCarrier wraps a []kafkago.Header for extracting OTel context on the consumer side.
type ExtractCarrier []kafkago.Header

func (c ExtractCarrier) Get(key string) string {
	for _, h := range c {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c ExtractCarrier) Set(_, _ string) {} // read-only on consumer side

func (c ExtractCarrier) Keys() []string {
	keys := make([]string, len(c))
	for i, h := range c {
		keys[i] = h.Key
	}
	return keys
}

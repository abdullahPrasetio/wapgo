package kafka

import (
	"testing"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

// ── kafkaHeaderCarrier ───────────────────────────────────────────────────────

func TestKafkaHeaderCarrier_SetAndGet(t *testing.T) {
	headers := []kafkago.Header{}
	c := &kafkaHeaderCarrier{headers: &headers}

	c.Set("traceparent", "00-abc-def-01")
	assert.Equal(t, "00-abc-def-01", c.Get("traceparent"))

	// Update existing key
	c.Set("traceparent", "00-xyz-uvw-01")
	assert.Equal(t, "00-xyz-uvw-01", c.Get("traceparent"))

	// Missing key returns empty string
	assert.Equal(t, "", c.Get("missing"))
}

func TestKafkaHeaderCarrier_Keys(t *testing.T) {
	headers := []kafkago.Header{
		{Key: "a", Value: []byte("1")},
		{Key: "b", Value: []byte("2")},
	}
	c := &kafkaHeaderCarrier{headers: &headers}
	keys := c.Keys()
	assert.ElementsMatch(t, []string{"a", "b"}, keys)
}

// ── ExtractCarrier ───────────────────────────────────────────────────────────

func TestExtractCarrier_Get(t *testing.T) {
	ec := ExtractCarrier([]kafkago.Header{
		{Key: "x-request-id", Value: []byte("req-1")},
	})
	assert.Equal(t, "req-1", ec.Get("x-request-id"))
	assert.Equal(t, "", ec.Get("not-there"))
}

func TestExtractCarrier_SetIsNoop(t *testing.T) {
	ec := ExtractCarrier([]kafkago.Header{})
	ec.Set("key", "value") // must not panic and must not modify
	assert.Equal(t, "", ec.Get("key"))
}

func TestExtractCarrier_Keys(t *testing.T) {
	ec := ExtractCarrier([]kafkago.Header{
		{Key: "k1", Value: []byte("v1")},
		{Key: "k2", Value: []byte("v2")},
	})
	assert.ElementsMatch(t, []string{"k1", "k2"}, ec.Keys())
}

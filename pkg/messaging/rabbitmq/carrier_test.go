package rabbitmq

import (
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestAMQPTableCarrier_Get_ExistingKey(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{"traceparent": "value-1"})
	assert.Equal(t, "value-1", c.Get("traceparent"))
}

func TestAMQPTableCarrier_Get_MissingKey(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{})
	assert.Equal(t, "", c.Get("missing"))
}

func TestAMQPTableCarrier_Get_NonStringValue(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{"key": 42})
	assert.Equal(t, "", c.Get("key"))
}

func TestAMQPTableCarrier_Set(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{})
	c.Set("traceparent", "00-abc-01")
	assert.Equal(t, "00-abc-01", c.Get("traceparent"))
}

func TestAMQPTableCarrier_Set_Overwrite(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{"k": "v1"})
	c.Set("k", "v2")
	assert.Equal(t, "v2", c.Get("k"))
}

func TestAMQPTableCarrier_Keys(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{"a": "1", "b": "2"})
	assert.ElementsMatch(t, []string{"a", "b"}, c.Keys())
}

func TestAMQPTableCarrier_Keys_Empty(t *testing.T) {
	c := amqpTableCarrier(amqp.Table{})
	assert.Empty(t, c.Keys())
}

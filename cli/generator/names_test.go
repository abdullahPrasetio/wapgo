package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNames_SnakeInput(t *testing.T) {
	n := NewNames("product_order")
	assert.Equal(t, "product_order", n.Snake)
	assert.Equal(t, "ProductOrder", n.Pascal)
	assert.Equal(t, "productOrder", n.Camel)
	assert.Equal(t, "product_orders", n.Table)
	assert.Equal(t, "product_order", n.Original)
}

func TestNewNames_KebabInput(t *testing.T) {
	n := NewNames("order-item")
	assert.Equal(t, "order_item", n.Snake)
	assert.Equal(t, "OrderItem", n.Pascal)
	assert.Equal(t, "orderItem", n.Camel)
	assert.Equal(t, "order_items", n.Table)
}

func TestNewNames_PascalInput(t *testing.T) {
	n := NewNames("ProductItem")
	assert.Equal(t, "product_item", n.Snake)
	assert.Equal(t, "ProductItem", n.Pascal)
	assert.Equal(t, "productItem", n.Camel)
}

func TestNewNames_SingleWord(t *testing.T) {
	n := NewNames("product")
	assert.Equal(t, "product", n.Snake)
	assert.Equal(t, "Product", n.Pascal)
	assert.Equal(t, "product", n.Camel)
	assert.Equal(t, "products", n.Table)
}

func TestToTable_YEnding(t *testing.T) {
	// "category" ends with 'y' preceded by consonant → "categories"
	n := NewNames("category")
	assert.Equal(t, "categories", n.Table)
}

func TestToTable_YEndingAfterVowel(t *testing.T) {
	// "key" ends with 'y' preceded by vowel 'e' → "keys"
	n := NewNames("key")
	assert.Equal(t, "keys", n.Table)
}

func TestToTable_SEnding(t *testing.T) {
	n := NewNames("class")
	assert.Equal(t, "classes", n.Table)
}

func TestToTable_XEnding(t *testing.T) {
	n := NewNames("box")
	assert.Equal(t, "boxes", n.Table)
}

func TestToTable_ShEnding(t *testing.T) {
	n := NewNames("dish")
	assert.Equal(t, "dishes", n.Table)
}

func TestNewNames_ModuleAndAppName(t *testing.T) {
	n := NewNames("user")
	n.Module = "github.com/me/svc"
	n.AppName = "my-svc"
	assert.Equal(t, "github.com/me/svc", n.Module)
	assert.Equal(t, "my-svc", n.AppName)
}

func TestNewNames_UpperCaseInput(t *testing.T) {
	n := NewNames("USER")
	assert.Equal(t, "user", n.Snake)
	assert.Equal(t, "User", n.Pascal)
}

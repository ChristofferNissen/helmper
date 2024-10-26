package state

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestGetValue(t *testing.T) {
	v := viper.New()
	v.Set("key", "value")

	// Test GetValue for string
	result := GetValue[string](v, "key")
	assert.Equal(t, "value", result)

	// Test GetValue for int
	v.Set("intKey", 42)
	intResult := GetValue[int](v, "intKey")
	assert.Equal(t, 42, intResult)
}

func TestSetValue(t *testing.T) {
	v := viper.New()

	// Test SetValue for string
	SetValue(v, "key", "value")
	result := v.GetString("key")
	assert.Equal(t, "value", result)

	// Test SetValue for int
	SetValue(v, "intKey", 42)
	intResult := v.GetInt("intKey")
	assert.Equal(t, 42, intResult)
}

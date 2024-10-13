package gcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCacheKeyTagKey(t *testing.T) {
	key := "test-key"
	expected := "key:test-key"
	actual := cacheKeyTagKey(key)
	assert.Equal(t, expected, actual)
}

func TestCacheKeyTagValue(t *testing.T) {
	key := "test-key"
	value := "test-value"
	expected := "value:test-key:test-value"
	actual := cacheKeyTagValue(key, value)
	assert.Equal(t, expected, actual)
}

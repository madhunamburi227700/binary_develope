package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaderXUserConstant(t *testing.T) {
	assert.Equal(t, "X-User", HeaderXUser, "HeaderXUser constant should be 'X-User'")
}

package anyx

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDeepZero(t *testing.T) {
	assert.Zero(t, DeepZero[int]())
	assert.Zero(t, *DeepZero[*int]())
	assert.Zero(t, DeepZero[time.Time]())
	assert.Zero(t, *DeepZero[*time.Time]())
}

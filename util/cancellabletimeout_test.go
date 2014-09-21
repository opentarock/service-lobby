package util_test

import (
	"testing"
	"time"

	"github.com/opentarock/service-lobby/util"
	"github.com/stretchr/testify/assert"
)

const timeoutDuration = 50 * time.Millisecond

func TestTimeoutFunctionIsCalledOnTimeout(t *testing.T) {
	called := false
	util.StartCancellableTimeout(timeoutDuration, func() {
		called = true
	})
	time.Sleep(timeoutDuration * 2)
	assert.True(t, called)
}

func TestTimeoutCanBeCancelled(t *testing.T) {
	called := false
	ct := util.StartCancellableTimeout(timeoutDuration, func() {
		called = true
	})
	ct.Cancel()
	time.Sleep(timeoutDuration * 2)
	assert.False(t, called)
}

func TestTimeoutCanBeCancelledMultipleTimes(t *testing.T) {
	called := false
	ct := util.StartCancellableTimeout(timeoutDuration, func() {
		called = true
	})
	ct.Cancel()
	ct.Cancel()
	time.Sleep(timeoutDuration * 2)
	assert.False(t, called)
}

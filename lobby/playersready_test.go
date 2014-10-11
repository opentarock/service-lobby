package lobby_test

import (
	"testing"
	"time"

	"github.com/opentarock/service-api/go/user"
	"github.com/opentarock/service-lobby/lobby"
	"github.com/stretchr/testify/assert"
)

var state = map[user.Id]string{
	user.Id("2"): "state2",
	user.Id("3"): "state3",
}

func MakePlayersReady() *lobby.PlayersReady {
	return lobby.NewPlayersReady(user.Id("1"), state, func() {})
}

const defaultTimeout = 10 * time.Second

func TestUniqueIdIsGenerated(t *testing.T) {
	pr1 := MakePlayersReady()
	pr2 := MakePlayersReady()
	assert.NotEqual(t, pr1.GetId(), pr2.GetId())
}

func TestHasUser(t *testing.T) {
	pr := MakePlayersReady()
	assert.True(t, pr.HasUser(user.Id("1")), "Owner is part of the ready process but is automatically ready.")
	assert.True(t, pr.HasUser(user.Id("2")))
	assert.True(t, pr.HasUser(user.Id("3")))
	assert.False(t, pr.HasUser(user.Id("4")))
}

func TestNoUserIsInitiallyReady(t *testing.T) {
	pr := MakePlayersReady()
	pr.Start(defaultTimeout, func(timeoutId string) {})
	defer pr.Cancel()
	assert.Equal(t, 0, pr.NumReady())
}

func TestPlayerCanBecomeReady(t *testing.T) {
	pr := MakePlayersReady()
	pr.Start(defaultTimeout, func(t string) {})
	defer pr.Cancel()
	err := pr.Ready(user.Id("2"), state[user.Id("2")])
	assert.Nil(t, err)
	assert.Equal(t, 1, pr.NumReady())
}

func TestPlayerWithWrongStateValueCantBecomeReady(t *testing.T) {
	pr := MakePlayersReady()
	pr.Start(defaultTimeout, func(t string) {})
	defer pr.Cancel()
	err := pr.Ready(user.Id("2"), "wrong")
	assert.Equal(t, lobby.ErrInvalidStateString, err)
	assert.Equal(t, 0, pr.NumReady())
}

func TestPlayerCantBecomeReadyMultipleTimes(t *testing.T) {
	pr := MakePlayersReady()
	pr.Start(defaultTimeout, func(t string) {})
	defer pr.Cancel()
	err := pr.Ready(user.Id("2"), state[user.Id("2")])
	assert.Nil(t, err)
	err = pr.Ready(user.Id("2"), state[user.Id("2")])
	assert.Equal(t, lobby.ErrAlreadyReady, err)
	assert.Equal(t, 1, pr.NumReady())
}

func TestUnknownPlayerCantGetReady(t *testing.T) {
	pr := MakePlayersReady()
	pr.Start(defaultTimeout, func(t string) {})
	defer pr.Cancel()
	err := pr.Ready(user.Id("10"), "state")
	assert.Equal(t, lobby.ErrUnknownUser, err)
	assert.Equal(t, 0, pr.NumReady())
}

func TestSuccessCallbackIsExecutedWhenAllThePLayersAreReady(t *testing.T) {
	ready := false
	pr := lobby.NewPlayersReady(user.Id("1"), state, func() {
		ready = true
	})
	pr.Start(defaultTimeout, func(t string) {})
	defer pr.Cancel()

	err := pr.Ready("2", state["2"])
	assert.Nil(t, err)

	err = pr.Ready("3", state["3"])
	assert.Nil(t, err)
	assert.Equal(t, 2, pr.NumReady())
	assert.True(t, ready, "Callback must set this to true")
}

func TestTimeoutCallbackIsExecuted(t *testing.T) {
	pr := MakePlayersReady()
	timeout := false
	pr.Start(100*time.Millisecond, func(t string) {
		timeout = true
	})
	time.Sleep(200 * time.Millisecond)
	assert.True(t, timeout, "Timeout callback should set it to true")
}

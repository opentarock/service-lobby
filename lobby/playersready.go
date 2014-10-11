package lobby

import (
	"errors"
	"sync"
	"time"

	"github.com/opentarock/service-api/go/user"
	"github.com/opentarock/service-lobby/util"

	"code.google.com/p/go-uuid/uuid"
)

// PlayersReady is a helper for managing ready status of players.
type PlayersReady struct {
	owner           user.Id
	playerState     map[user.Id]string
	playersReady    map[user.Id]bool
	timeout         *util.CancellableTimeout
	id              string
	successCallback func()
	lock            *sync.Mutex
}

// NewPlayersReady return a new PlayersReady  with expected player state and callback function f
// that is executed when all te players are ready.
func NewPlayersReady(owner user.Id, playerState map[user.Id]string, f func()) *PlayersReady {
	id := uuid.New()
	ready := &PlayersReady{
		owner:           owner,
		playerState:     playerState,
		playersReady:    make(map[user.Id]bool),
		id:              id,
		successCallback: f,
		lock:            new(sync.Mutex),
	}
	return ready
}

// Start starts a process of collecting player's ready status.
func (r *PlayersReady) Start(timeout time.Duration, f func(timeoutId string)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.timeout = util.StartCancellableTimeout(timeout, func() {
		f(r.id)
	})
}

// GetId returns a unique id.
func (r *PlayersReady) GetId() string {
	return r.id
}

// NumReady return the number of users that are currently ready.
func (r *PlayersReady) NumReady() uint {
	r.lock.Lock()
	defer r.lock.Unlock()
	return uint(len(r.playersReady))
}

// ErrAlreadyReady is returned by Ready when a user trying to become ready is
// already ready.
var ErrAlreadyReady = errors.New("User already ready.")

// ErrorInvalidStateString is returned by Ready when the state does not match
// the state that is expected.
var ErrInvalidStateString = errors.New("Invalid state string.")

// EeeunknownUser is returned by Ready if the user trying to become ready is not
// part of this ready process.
var ErrUnknownUser = errors.New("unknown user.")

// Ready marks a user with user id as ready if the state matches the expected state
// for that user.
// If all the users are ready room's game is started.
func (r *PlayersReady) Ready(userId user.Id, stateReceived string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.playersReady[userId]; ok {
		return ErrAlreadyReady
	}
	if state, ok := r.playerState[userId]; ok {
		if state == stateReceived {
			r.playersReady[userId] = true
		} else {
			return ErrInvalidStateString
		}
	} else {
		return ErrUnknownUser
	}
	if len(r.playerState) == len(r.playersReady) {
		r.timeout.Cancel()
		r.successCallback()
	}
	return nil
}

// HasUser checks if a user is part of this ready process.
func (r *PlayersReady) HasUser(userId user.Id) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	_, ok := r.playerState[userId]
	return ok || userId == r.owner
}

// Cancel does cancel this ready process' ready timeout.
// Users can still become ready with the call to Ready just without a timeout.
func (r *PlayersReady) Cancel() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.timeout.Cancel()
}

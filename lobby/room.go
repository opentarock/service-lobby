package lobby

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"code.google.com/p/go-uuid/uuid"
	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-lobby/util"
)

// stateLength is the default length of generated player state token string.
const stateLength = 32

// readyTimeout is the default timeout duration for players to confirm that they are ready.
const readyTimeout = time.Second * 15

// roomStatus represents the current status of game in the room.
type roomStatus int

const (
	notStarted roomStatus = iota
	starting
	inProgress
)

// Room represents a game room allowing joining and leaving of users.
// All the methods on room are thread safe.
type Room struct {
	id           RoomId
	name         string
	options      *proto_lobby.RoomOptions
	owner        uint64
	maxPlayers   uint
	players      map[uint64]string
	status       roomStatus
	ready        *PlayersReady
	ReadyTimeout time.Duration
	lock         *sync.Mutex // Because room pointers are shared we must own a lock before reading or writing its data.
}

// NewRoom returns a new Room with a given name, owner and max players allowed.
// Room id is automatically generated and can be accessed using GetId().
func NewRoom(name string, owner uint64, maxPlayers uint) *Room {
	return &Room{
		id:           newRoomId(),
		name:         name,
		owner:        owner,
		maxPlayers:   maxPlayers,
		players:      make(map[uint64]string),
		status:       notStarted,
		ReadyTimeout: readyTimeout,
		lock:         new(sync.Mutex),
	}
}

// newRoomId generates a unique id.
func newRoomId() RoomId {
	return RoomId(uuid.New())
}

// GetId returns a unique identifier for the room.
func (r *Room) GetId() RoomId {
	return r.id
}

// IsInProgress returns true if the game in the room is in progress.
func (r *Room) IsInProgress() bool {
	return r.status == inProgress
}

// IsStarting return true is the game in the room is starting.
func (r *Room) IsStarting() bool {
	return r.status == starting
}

// IsStarted return true if the room game is in progress or is in the process
// of collecting player ready sttaus.
func (r *Room) IsStarted() bool {
	return r.IsInProgress() || r.IsStarting()
}

// Join adds a user to the room.
// If the room is full error is returned. If user with this id is already in
// the room Join is a NOOP.
// User can join the room regardless of the room's current status. If the player
// ready process is in progress joined user is automatically considered ready.
func (r *Room) Join(userId uint64) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.numPlayers() == r.maxPlayers {
		return fmt.Errorf("Room is full (maxPlayers=%d)", r.maxPlayers)
	}
	r.players[userId] = util.RandomToken(stateLength)
	return nil
}

// ErrGameStartInProgress is returned by Leave if the user tries to leave the
// room in the middle of starting the game.
var ErrGameStartInProgress = errors.New("Game start in progress")

// Leave removes a user from the room.
// If the user leaving is the last one in the room, owner is set to 0 and false
// is returned indicating that there are no more players in the room.
// If the game start is in progress the user that is part of the player ready process
// can't leave the room untill the process is not complete.
func (r *Room) Leave(userId uint64) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.ready != nil && r.ready.HasUser(userId) {
		return true, ErrGameStartInProgress
	}
	if r.numPlayers() == 1 {
		r.owner = 0
		return false, nil
	}
	// If the user leaving is the owner we choose a new one.
	if r.owner == userId {
		r.owner = takeOne(r.players)
		delete(r.players, r.owner)
	} else {
		delete(r.players, userId)
	}
	return true, nil
}

// takeOne takes one key from the input map and returns it.
// If the map is empty zero is returned. There is no specified order in which
// the key is selected.
func takeOne(m map[uint64]string) uint64 {
	for key, _ := range m {
		return key
	}
	return 0
}

// Getowner return the current room owner user id.
func (r *Room) GetOwner() uint64 {
	return r.owner
}

// GetUserIds returns user ids of all the players currently in the room.
func (r *Room) GetUserIds() []uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	result := r.getNonOwnerUserIdsHelper()
	result = append(result, r.owner)
	return result
}

// GetNonOwnerUserIds returns user ids of all the players currently in the room
// except the owner.
func (r *Room) GetNonOwnerUserIds() []uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.getNonOwnerUserIdsHelper()
}

// getNonOwnerUserIdshelper is a helper function that returns user ids of all
// players currently in the room except the owner without claiming any locks.
func (r *Room) getNonOwnerUserIdsHelper() []uint64 {
	// A little optimization because we use this method in GetUserIds so we
	// don't have to reallocate.
	result := make([]uint64, 0, 1+len(r.players))
	for userId, _ := range r.players {
		result = append(result, userId)
	}
	return result
}

// NumPlayers return the current number of players in the room.
func (r *Room) NumPlayers() uint {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.numPlayers()
}

func (r *Room) numPlayers() uint {
	return uint(1 + len(r.players))
}

// Proto converts a Room internal representation to a Protobuf representation
// suitable for sending as a service response.
// TODO: needs testing
func (r *Room) Proto() *proto_lobby.Room {
	r.lock.Lock()
	defer r.lock.Unlock()
	return &proto_lobby.Room{
		Id:      pbuf.String(r.id.String()),
		Name:    &r.name,
		Options: r.options,
		Owner:   fetchPlayerInfo(r.owner),
		Players: fetchPlayersInfo(r.players),
	}
}

// TODO: real implementation should get nicknames from user service.
func fetchPlayerInfo(userId uint64) *proto_lobby.Player {
	return &proto_lobby.Player{
		UserId:   &userId,
		Nickname: pbuf.String(fmt.Sprintf("nickname %d", userId)),
	}
}

func fetchPlayersInfo(userIds map[uint64]string) []*proto_lobby.Player {
	playerInfoList := make([]*proto_lobby.Player, 0, len(userIds))
	for userId, _ := range userIds {
		playerInfoList = append(playerInfoList, fetchPlayerInfo(userId))
	}
	return playerInfoList
}

// ErrAlready started is returned by StartGame if the room game is already in progress
// or starting.
var ErrAlreadyStarted = errors.New("Room game already started.")

// StartGame begins a process of players confirming they are ready to start a game.
// When the ready process is complete game is started.
// A map of random states for every user is returned that should be sent to
// the matching users so they can confirm they are ready.
func (r *Room) StartGame() (map[uint64]string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.status != notStarted {
		return nil, ErrAlreadyStarted
	}
	if r.numPlayers() == 1 {
		r.finishStartGame()
	} else {
		r.ready = NewPlayersReady(r.owner, copyMap(r.players), func() {
			// Lock needed for this method is claimed in PlayerReady.
			r.finishStartGame()
		})
		r.ready.Start(r.ReadyTimeout, func(timeoutId string) {
			r.resetRoomStatus(timeoutId)
		})
		r.status = starting
	}
	return copyMap(r.players), nil
}

// ErrNotStarted is returned by CancelStart if the game is not in the process of starting.
var ErrNotStarting = errors.New("Room game not starting.")

// CancelStart cancels the start of the game and resets the room's status.
func (r *Room) CancelStart() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.status != starting {
		return ErrNotStarting
	}
	r.reset()
	return nil
}

// ErrUnexpectedReady is returned by PlayerReady if room status is not starting.
var ErrUnexpectedReady = errors.New("Unexpected ready.")

// PlayerReady marks the user with given id and state as ready.
func (r *Room) PlayerReady(userId uint64, state string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.status != starting {
		return ErrUnexpectedReady
	}
	return r.ready.Ready(userId, state)
}

// finishStartGame starts the game and updates the room's status.
// This method should only be called if lock to this room is currently owned.
// TODO: game should be started
func (r *Room) finishStartGame() {
	r.status = inProgress
	log.Printf("All players in room [id=%s] are ready, starting game.", r.id)
}

// resetRoomSttaus resets the room status to it's initial values.
// New random player state is generated so outdated PlayerReady requests are
// rejected.
// Timeout id must match the id of the user ready process helper, if it does not
// nothing happens.
func (r *Room) resetRoomStatus(timeoutId string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.ready != nil && r.ready.GetId() == timeoutId {
		r.reset()
	}
}

// reset resets room's status without claiming any locks.
func (r *Room) reset() {
	r.status = notStarted
	r.ready.Cancel()
	r.ready = nil
	for userId, _ := range r.players {
		r.players[userId] = util.RandomToken(stateLength)
	}
}

// copyMap makes a copy of a map and returns it.
func copyMap(m map[uint64]string) map[uint64]string {
	r := make(map[uint64]string)
	for k, v := range m {
		r[k] = v
	}
	return r
}

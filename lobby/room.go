package lobby

import (
	"fmt"
	"sync"

	"code.google.com/p/go-uuid/uuid"
	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_lobby"
)

// Room represents a game room allowing joining and leaving of users.
// All the methods on room are thread safe.
type Room struct {
	id         RoomId
	name       string
	options    *proto_lobby.RoomOptions
	owner      uint64
	maxPlayers uint
	players    []uint64
	lock       *sync.Mutex // Because room pointers are shared we must own a lock before reading or writing its data.
}

// NewRoom returns a new Room with a given name, owner and max players allowed.
// Room id is automatically generated and can be accessed using GetId().
func NewRoom(name string, owner uint64, maxPlayers uint) *Room {
	return &Room{
		id:         newRoomId(),
		name:       name,
		owner:      owner,
		maxPlayers: maxPlayers,
		players:    make([]uint64, 0, maxPlayers-1),
		lock:       new(sync.Mutex),
	}
}

func newRoomId() RoomId {
	return RoomId(uuid.New())
}

// GetId returns a unique identifier for the room.
func (r *Room) GetId() RoomId {
	return r.id
}

// Join adds a user to the room.
// If the room is full error is returned. If user with this id is already in
// the room Join is a NOOP.
func (r *Room) Join(userId uint64) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.NumPlayers() == r.maxPlayers {
		return fmt.Errorf("Room is full (maxPlayers=%d)", r.maxPlayers)
	}
	if !containsElem(r.players, userId) {
		r.players = append(r.players, userId)
	}
	return nil
}

// containsElem checks if a value is present in the input slice.
func containsElem(s []uint64, x uint64) bool {
	for _, elem := range s {
		if elem == x {
			return true
		}
	}
	return false
}

// Leave removes a user from the room.
// If the user leaving is the last one in the room, owner is set to 0 and false
// is returned indicating that there are no more players in the room.
func (r *Room) Leave(userId uint64) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.NumPlayers() == 1 {
		r.owner = 0
		return false
	}
	// If the user leaving is the owner we choose a new one.
	if r.owner == userId {
		r.owner = r.players[0]
		r.players = r.players[1:]
	} else {
		r.players = removeElem(r.players, userId)
	}
	return true
}

// removeElem removes an element from a slice and returns a new slice without
// that element.
// New slice is always returned regardless of input slice containing the element or not.
func removeElem(s []uint64, x uint64) []uint64 {
	result := make([]uint64, 0)
	for _, elem := range s {
		if elem != x {
			result = append(result, elem)
		}
	}
	return result
}

// Getowner return the current room owner user id.
func (r *Room) GetOwner() uint64 {
	return r.owner
}

// GetuserIds returns user ids of all the players currently in the room.
func (r *Room) GetUserIds() []uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	result := make([]uint64, 1+len(r.players))
	result[0] = r.owner
	copy(result[1:], r.players)
	return result
}

// NumPlayers return the current number of players in the room.
func (r *Room) NumPlayers() uint {
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

func fetchPlayersInfo(userIds []uint64) []*proto_lobby.Player {
	playerInfoList := make([]*proto_lobby.Player, 0, len(userIds))
	for _, userId := range userIds {
		playerInfoList = append(playerInfoList, fetchPlayerInfo(userId))
	}
	return playerInfoList
}

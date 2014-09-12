package lobby

import (
	"fmt"
	"log"
	"sync"

	"code.google.com/p/go-uuid/uuid"
	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto_lobby"
)

type RoomId string

func (r RoomId) String() string {
	return string(r)
}

type room struct {
	id      RoomId
	name    string
	options *proto_lobby.RoomOptions
	owner   uint64
	players []uint64
	lock    *sync.Mutex
}

func (r *room) Proto() *proto_lobby.Room {
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

type Rooms map[RoomId]*room

type Players map[uint64]*room

type RoomList struct {
	rooms     Rooms
	roomsLock *sync.RWMutex
}

func NewRoomList() *RoomList {
	return &RoomList{
		rooms:     make(Rooms),
		roomsLock: new(sync.RWMutex),
	}
}

func (r *RoomList) CreateRoom(
	userId uint64, roomName string, options *proto_lobby.RoomOptions) *proto_lobby.Room {

	log.Printf("User [id=%d] created a room [name=%s]", userId, roomName)
	roomId := newRoomId()
	room := &room{
		id:      roomId,
		name:    roomName,
		options: options,
		owner:   userId,
		players: make([]uint64, 0, 4),
		lock:    new(sync.Mutex),
	}
	r.roomsLock.Lock()
	defer r.roomsLock.Unlock()
	r.rooms[roomId] = room
	return room.Proto()
}

func newRoomId() RoomId {
	return RoomId(uuid.New())
}

func (r *RoomList) JoinRoom(userId uint64, roomId RoomId) *proto_lobby.Room {
	room := r.findRoom(roomId)
	joinRoom(room, userId)
	return room.Proto()
}

func joinRoom(room *room, userId uint64) {
	log.Printf("User [id=%d] joined room [name=%s]", userId, room.name)
	room.lock.Lock()
	defer room.lock.Unlock()
	room.players = append(room.players, userId)
}

func (r *RoomList) ListRoomsExcluding(userId uint64) []*proto_lobby.Room {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	roomList := make([]*proto_lobby.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		if room.owner == userId {
			continue
		}
		roomList = append(roomList, room.Proto())
	}
	return roomList
}

func (r *RoomList) findRoom(roomId RoomId) *room {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	return r.rooms[roomId]
}

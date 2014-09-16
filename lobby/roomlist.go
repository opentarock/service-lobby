package lobby

import (
	"fmt"
	"log"
	"sync"

	"code.google.com/p/go-uuid/uuid"
	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_lobby"
)

type RoomId string

func (r RoomId) String() string {
	return string(r)
}

type room struct {
	id         RoomId
	name       string
	options    *proto_lobby.RoomOptions
	owner      uint64
	maxPlayers uint
	players    []uint64
	lock       *sync.Mutex
}

func (r *room) Proto() *proto_lobby.Room {
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

func (r *room) GetUserIds() []uint64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	result := make([]uint64, 1+len(r.players))
	result[0] = r.owner
	copy(result[1:], r.players)
	return result
}

func (r *room) CountPlayers() uint {
	return uint(1 + len(r.players))
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

type Players map[uint64]RoomId

type RoomList struct {
	rooms        Rooms
	roomsLock    *sync.RWMutex
	players      Players
	playersLock  *sync.RWMutex
	notifyClient client.NotifyClient
}

func NewRoomList(notifyClient client.NotifyClient) *RoomList {
	return &RoomList{
		rooms:        make(Rooms),
		roomsLock:    new(sync.RWMutex),
		players:      make(Players),
		playersLock:  new(sync.RWMutex),
		notifyClient: notifyClient,
	}
}

func (r *RoomList) CreateRoom(
	userId uint64,
	roomName string,
	options *proto_lobby.RoomOptions) (*proto_lobby.Room, proto_lobby.CreateRoomResponse_ErrorCode) {

	if r.isPlayerInRoom(userId) {
		return nil, proto_lobby.CreateRoomResponse_ALREADY_IN_ROOM
	}

	roomId := newRoomId()
	log.Printf("User [id=%d] created a room [id=%s]", userId, roomId)
	room := &room{
		id:         roomId,
		name:       roomName,
		options:    options,
		owner:      userId,
		maxPlayers: 4,
		players:    make([]uint64, 0, 4),
		lock:       new(sync.Mutex),
	}
	r.setPlayerRoom(userId, roomId)
	r.roomsLock.Lock()
	defer r.roomsLock.Unlock()
	r.rooms[roomId] = room
	return room.Proto(), 0
}

func newRoomId() RoomId {
	return RoomId(uuid.New())
}

func (r *RoomList) JoinRoom(
	userId uint64,
	roomId RoomId) (*proto_lobby.Room, proto_lobby.JoinRoomResponse_ErrorCode) {

	room := r.findRoom(roomId)
	if r.isPlayerInRoom(userId) {
		leaveRoom(room, userId)
	}
	if room == nil {
		return nil, proto_lobby.JoinRoomResponse_ROOM_DOES_NOT_EXIST
	} else if room.CountPlayers() == room.maxPlayers {
		return nil, proto_lobby.JoinRoomResponse_ROOM_FULL
	}
	r.setPlayerRoom(userId, roomId)
	usersInRoom := room.GetUserIds()
	joinRoom(room, userId)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: fetchPlayerInfo(userId),
	}, usersInRoom...)
	return room.Proto(), 0
}

func joinRoom(room *room, userId uint64) {
	room.lock.Lock()
	defer room.lock.Unlock()
	room.players = append(room.players, userId)
	log.Printf("User [id=%d] joined room [id=%s]", userId, room.id)
}

func (r *RoomList) LeaveRoom(userId uint64) (bool, proto_lobby.LeaveRoomResponse_ErrorCode) {
	roomId := r.findPlayerRoom(userId)
	room := r.findRoom(roomId)
	if room == nil {
		return false, proto_lobby.LeaveRoomResponse_NOT_IN_ROOM
	}
	r.removePlayerRoom(userId)
	leaveRoom(room, userId)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: fetchPlayerInfo(userId),
	}, room.GetUserIds()...)
	return true, 0
}

func leaveRoom(room *room, userId uint64) {
	room.lock.Lock()
	defer room.lock.Unlock()
	if room.owner == userId {
		room.owner = room.players[0]
		room.players = room.players[1:]
	} else {
		room.players = removeElem(room.players, userId)
	}
	log.Printf("User [id=%d] left room [id=%s]", userId, room.id)
}

func removeElem(s []uint64, x uint64) []uint64 {
	result := make([]uint64, 0)
	for _, elem := range s {
		if elem != x {
			result = append(result, elem)
		}
	}
	return result
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

func (r *RoomList) GetRoom(roomId RoomId) *proto_lobby.Room {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	room := r.findRoom(roomId)
	if room != nil {
		return room.Proto()
	}
	return nil
}

func (r *RoomList) findRoom(roomId RoomId) *room {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	return r.rooms[roomId]
}

func (r *RoomList) findPlayerRoom(userId uint64) RoomId {
	r.playersLock.RLock()
	defer r.playersLock.RUnlock()
	return r.players[userId]
}

func (r *RoomList) isPlayerInRoom(userId uint64) bool {
	return r.findPlayerRoom(userId) != ""
}

func (r *RoomList) setPlayerRoom(userId uint64, roomId RoomId) {
	r.playersLock.Lock()
	defer r.playersLock.Unlock()
	r.players[userId] = roomId
}

func (r *RoomList) removePlayerRoom(userId uint64) {
	r.playersLock.Lock()
	defer r.playersLock.Unlock()
	delete(r.players, userId)
}

func (r *RoomList) notifyAsync(msg proto.ProtobufMessage, users ...uint64) {
	go func() {
		// TODO: handle response
		_, err := r.notifyClient.MessageUsers(msg, users...)
		if err != nil {
			log.Printf("Error sensing notifictations to clients: %s", err)
		}
	}()
}

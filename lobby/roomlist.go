package lobby

import (
	"log"
	"sync"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_lobby"
)

type RoomId string

func (r RoomId) String() string {
	return string(r)
}

type Rooms map[RoomId]*Room

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

	room := NewRoom(roomName, userId, 4)
	r.setPlayerRoom(userId, room.id)
	r.roomsLock.Lock()
	defer r.roomsLock.Unlock()
	r.rooms[room.id] = room
	log.Printf("User [id=%d] created a room [id=%s]", userId, room.id)
	return room.Proto(), 0
}

func (r *RoomList) JoinRoom(
	userId uint64,
	roomId RoomId) (*proto_lobby.Room, proto_lobby.JoinRoomResponse_ErrorCode) {

	room := r.findRoom(roomId)
	if r.isPlayerInRoom(userId) {
		r.LeaveRoom(userId)
	}
	if room == nil {
		return nil, proto_lobby.JoinRoomResponse_ROOM_DOES_NOT_EXIST
	}
	usersInRoom := room.GetUserIds()
	r.setPlayerRoom(userId, roomId)
	if err := room.Join(userId); err != nil {
		return nil, proto_lobby.JoinRoomResponse_ROOM_FULL
	}
	log.Printf("User [id=%d] joined room [id=%s]", userId, room.id)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: fetchPlayerInfo(userId),
	}, usersInRoom...)
	return room.Proto(), 0
}

func (r *RoomList) LeaveRoom(userId uint64) (bool, proto_lobby.LeaveRoomResponse_ErrorCode) {
	roomId := r.findPlayerRoom(userId)
	room := r.findRoom(roomId)
	if room == nil {
		return false, proto_lobby.LeaveRoomResponse_NOT_IN_ROOM
	}
	r.removePlayerRoom(userId)
	if notEmpty := room.Leave(userId); !notEmpty {
		r.removeRoom(roomId)
	}
	log.Printf("User [id=%d] left room [id=%s]", userId, room.id)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: fetchPlayerInfo(userId),
	}, room.GetUserIds()...)
	return true, 0
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

func (r *RoomList) findRoom(roomId RoomId) *Room {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	return r.rooms[roomId]
}

func (r *RoomList) removeRoom(roomId RoomId) {
	r.roomsLock.RLock()
	defer r.roomsLock.RUnlock()
	delete(r.rooms, roomId)
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

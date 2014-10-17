package lobby

import (
	"errors"
	"log"
	"sync"

	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-api/go/user"
)

type RoomId string

func (r RoomId) String() string {
	return string(r)
}

type Rooms map[RoomId]*Room

type Players map[user.Id]RoomId

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
	userId user.Id,
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
	log.Printf("User [id=%s] created a room [id=%s]", userId, room.id)
	return room.Proto(), 0
}

func (r *RoomList) JoinRoom(
	userId user.Id,
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
	log.Printf("User [id=%s] joined room [id=%s]", userId, room.id)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: pbuf.String(userId.String()),
	}, usersInRoom...)
	return room.Proto(), 0
}

func (r *RoomList) LeaveRoom(userId user.Id) (bool, proto_lobby.LeaveRoomResponse_ErrorCode) {
	roomId := r.findPlayerRoom(userId)
	room := r.findRoom(roomId)
	if room == nil {
		return false, proto_lobby.LeaveRoomResponse_NOT_IN_ROOM
	}
	r.removePlayerRoom(userId)
	// TODO: handle error
	if notEmpty, _ := room.Leave(userId); !notEmpty {
		r.removeRoom(roomId)
	}
	log.Printf("User [id=%s] left room [id=%s]", userId, room.id)
	r.notifyAsync(&proto_lobby.JoinRoomEvent{
		Player: pbuf.String(userId.String()),
	}, room.GetUserIds()...)
	return true, 0
}

func (r *RoomList) ListRoomsExcluding(userId user.Id) []*proto_lobby.Room {
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

var (
	ErrNotInRoom = errors.New("User must create a room before starting the game")
	ErrNotOwner  = errors.New("Only owner can start the game")
)

func (r *RoomList) StartGame(userId user.Id) error {
	if !r.isPlayerInRoom(userId) {
		return ErrNotInRoom
	}
	room := r.getPlayerRoom(userId)
	if room.GetOwner() != userId {
		return ErrNotOwner
	}
	log.Printf("Owner [id=%s] started the game in room [id=%s]", userId, room.GetId())
	userState, err := room.StartGame()
	if err != nil {
		return err
	}
	r.notifyGameStart(room, userState)
	return nil
}

func (r *RoomList) notifyGameStart(room *Room, userState map[user.Id]string) {
	for _, userId := range room.GetNonOwnerUserIds() {
		log.Printf("State for user [id=%s] is %s", userId, userState[userId])
		r.notifyAsync(&proto_lobby.StartGameEvent{
			RoomId: pbuf.String(room.GetId().String()),
			State:  pbuf.String(userState[userId]),
		}, userId)
	}
}

func (r *RoomList) PlayerReady(userId user.Id, state string) error {
	if !r.isPlayerInRoom(userId) {
		return ErrNotInRoom
	}
	room := r.getPlayerRoom(userId)
	err := room.PlayerReady(userId, state)
	if err != nil {
		return err
	}
	log.Printf("Player [id=%s] in room [id=%s] is ready", userId, room.GetId())
	r.notifyPlayerReady(room, userId)
	return nil
}

func (r *RoomList) notifyPlayerReady(room *Room, readyUserId user.Id) {
	for _, userId := range room.GetUserIds() {
		if userId == readyUserId {
			continue
		}
		r.notifyAsync(&proto_lobby.PlayerReadyEvent{
			UserId: pbuf.String(userId.String()),
		}, userId)
	}
}

func (r *RoomList) getPlayerRoom(userId user.Id) *Room {
	return r.findRoom(r.findPlayerRoom(userId))
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

func (r *RoomList) findPlayerRoom(userId user.Id) RoomId {
	r.playersLock.RLock()
	defer r.playersLock.RUnlock()
	return r.players[userId]
}

func (r *RoomList) isPlayerInRoom(userId user.Id) bool {
	return r.findPlayerRoom(userId) != ""
}

func (r *RoomList) setPlayerRoom(userId user.Id, roomId RoomId) {
	r.playersLock.Lock()
	defer r.playersLock.Unlock()
	r.players[userId] = roomId
}

func (r *RoomList) removePlayerRoom(userId user.Id) {
	r.playersLock.Lock()
	defer r.playersLock.Unlock()
	delete(r.players, userId)
}

func (r *RoomList) notifyAsync(msg proto.ProtobufMessage, users ...user.Id) {
	go func() {
		// TODO: handle response
		//_, err := r.notifyClient.MessageUsers(msg, users...)
		//if err != nil {
		//log.Printf("Error sending notifictations to clients: %s", err)
		//}
	}()
}

package service

import (
	"log"
	"sync"

	"code.google.com/p/go-uuid/uuid"

	pbuf "code.google.com/p/gogoprotobuf/proto"

	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_headers"
	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-api/go/service"
)

type RoomId string

func (id RoomId) String() string {
	return string(id)
}

type Room struct {
	Name    string
	Options *proto_lobby.RoomOptions
	Owner   uint64
	Players []uint64
}

type lobbyServiceHandlers struct {
	roomMap  map[RoomId]*Room
	roomLock *sync.RWMutex
}

func NewLobbyServiceHandlers() *lobbyServiceHandlers {
	return &lobbyServiceHandlers{
		roomMap:  make(map[RoomId]*Room),
		roomLock: new(sync.RWMutex),
	}
}

func (s *lobbyServiceHandlers) CreateRoomHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) *proto.Message {
		var request proto_lobby.CreateRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return nil
		}
		var authHeader proto_headers.AuthorizationHeader
		_, err = msg.Header.Unmarshal(&authHeader)
		if err != nil {
			log.Println(err)
			return nil
		}
		s.addRoom(authHeader.GetUserId(), request.GetName(), request.GetOptions())
		response := proto_lobby.CreateRoomResponse{
			Name:    pbuf.String(request.GetName()),
			Options: request.GetOptions(),
		}
		responseMsg, err := proto.Marshal(&response)
		if err != nil {
			log.Println(err)
			return nil
		}
		return responseMsg
	})
}

func (s *lobbyServiceHandlers) addRoom(userId uint64, name string, options *proto_lobby.RoomOptions) {
	log.Printf("User [id=%d] created a room [name=%s]", userId, name)
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	s.roomMap[newRoomId()] = &Room{
		Name:    name,
		Options: options,
	}
}

func newRoomId() RoomId {
	return RoomId(uuid.New())
}

func (s *lobbyServiceHandlers) ListRoomsHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) *proto.Message {
		var request proto_lobby.ListRoomsRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return nil
		}
		response := proto_lobby.ListRoomsResponse{
			Rooms: s.listRooms(),
		}
		responseMsg, err := proto.Marshal(&response)
		if err != nil {
			log.Println(err)
			return nil
		}
		return responseMsg
	})
}

func (s *lobbyServiceHandlers) listRooms() []*proto_lobby.Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	roomList := make([]*proto_lobby.Room, 0, len(s.roomMap))
	for _, room := range s.roomMap {
		roomList = append(roomList, &proto_lobby.Room{
			Name:    &room.Name,
			Options: room.Options, // TODO: make a copy without password
			Owner:   makePlayer(),
			Players: []*proto_lobby.Player{makePlayer(), makePlayer()},
		})
	}
	return roomList
}

func makePlayer() *proto_lobby.Player {
	return &proto_lobby.Player{
		UserId:   pbuf.Uint64(1),
		Nickname: pbuf.String("some nickname"),
	}
}

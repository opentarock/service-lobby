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
		createRoomRequest, err := proto_lobby.AsCreateRoomRequest(msg)
		if err != nil {
			log.Println(err)
			return nil
		}
		authHeader, err := proto_headers.GetAuthorizationHeader(msg)
		if err != nil {
			log.Println(err)
			return nil
		} else if authHeader == nil {
			log.Println("Missing required header: AuthorizationHeader")
			return nil
		}
		s.addRoom(authHeader.GetUserId(), createRoomRequest.GetName(), createRoomRequest.GetOptions())
		response := proto_lobby.CreateRoomResponse{
			Name:    pbuf.String(createRoomRequest.GetName()),
			Options: createRoomRequest.GetOptions(),
		}
		responseData, err := pbuf.Marshal(&response)
		if err != nil {
			log.Println(err)
			return nil
		}
		responseMsg := proto.NewMessage(proto_lobby.CreateRoomResponseMessage, responseData)
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
		_, err := proto_lobby.AsListRoomsRequest(msg)
		if err != nil {
			log.Println(err)
			return nil
		}
		response := proto_lobby.ListRoomsResponse{
			Rooms: s.listRooms(),
		}
		responseData, err := pbuf.Marshal(&response)
		if err != nil {
			log.Println(err)
			return nil
		}
		responseMsg := proto.NewMessage(proto_lobby.ListRoomsResponseMessage, responseData)
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

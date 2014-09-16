package service

import (
	"log"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_headers"
	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-api/go/service"
	"github.com/opentarock/service-lobby/lobby"
)

type lobbyServiceHandlers struct {
	roomList *lobby.RoomList
}

func NewLobbyServiceHandlers(notifyClient client.NotifyClient) *lobbyServiceHandlers {
	return &lobbyServiceHandlers{
		roomList: lobby.NewRoomList(notifyClient),
	}
}

func WithAuth(h func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage) service.MessageHandler {

	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		var authHeader proto_headers.AuthorizationHeader
		_, err := msg.Header.Unmarshal(&authHeader)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}
		return h(&authHeader, msg)
	})
}

func (s *lobbyServiceHandlers) CreateRoomHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.CreateRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}
		room, errCode := s.roomList.CreateRoom(auth.GetUserId(), request.GetName(), request.GetOptions())
		response := proto_lobby.CreateRoomResponse{
			Room: room,
		}
		if room == nil {
			response.ErrorCode = errCode.Enum()
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) JoinRoomHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.JoinRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}

		room, errCode := s.roomList.JoinRoom(auth.GetUserId(), lobby.RoomId(request.GetRoomId()))

		response := proto_lobby.JoinRoomResponse{
			Room: room,
		}
		if room == nil {
			response.ErrorCode = errCode.Enum()
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) LeaveRoomHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.LeaveRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}

		success, errCode := s.roomList.LeaveRoom(auth.GetUserId())

		response := proto_lobby.LeaveRoomResponse{}
		if !success {
			response.ErrorCode = errCode.Enum()
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) ListRoomsHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.ListRoomsRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}
		response := proto_lobby.ListRoomsResponse{
			Rooms: s.roomList.ListRoomsExcluding(auth.GetUserId()),
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) RoomInfoHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.RoomInfoRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{}
		}
		response := proto_lobby.RoomInfoResponse{
			Room: s.roomList.GetRoom(lobby.RoomId(request.GetRoomId())),
		}
		if response.Room == nil {
			response.ErrorCode = proto_lobby.RoomInfoResponse_ROOM_DOES_NOT_EXIST.Enum()
		}
		return proto.CompositeMessage{Message: &response}
	})
}

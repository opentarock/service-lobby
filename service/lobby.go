package service

import (
	"log"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_errors"
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
		found, err := msg.Header.Unmarshal(&authHeader)
		if err != nil {
			log.Println(err)
			var msg proto.ProtobufMessage
			if found {
				msg = proto_errors.NewMalformedMessageUnpack()
			} else {
				msg = proto_errors.NewMissingHeader(authHeader.GetMessageType())
			}
			return proto.CompositeMessage{Message: msg}
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
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
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
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
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
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
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
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
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
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
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

func (s *lobbyServiceHandlers) StartGameHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.StartGameRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
		}
		err = s.roomList.StartGame(auth.GetUserId())
		var errResponse *proto_lobby.StartGameResponse_ErrorCode
		if err == lobby.ErrNotInRoom {
			errResponse = proto_lobby.StartGameResponse_NOT_IN_ROOM.Enum()
		} else if err == lobby.ErrNotOwner {
			errResponse = proto_lobby.StartGameResponse_NOT_OWNER.Enum()
		} else if err == lobby.ErrAlreadyStarted {
			errResponse = proto_lobby.StartGameResponse_ALREADY_STARTED.Enum()
		} else if err != nil {
			log.Println("Unknown error: %s", err)
			return proto.CompositeMessage{Message: proto_errors.NewInternalErrorUnknown()}
		}
		response := proto_lobby.StartGameResponse{
			ErrorCode: errResponse,
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) PlayerReadyHandler() service.MessageHandler {
	return WithAuth(func(auth *proto_headers.AuthorizationHeader, msg *proto.Message) proto.CompositeMessage {
		var request proto_lobby.PlayerReadyRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			log.Println(err)
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
		}
		err = s.roomList.PlayerReady(auth.GetUserId(), request.GetState())
		var errResponse *proto_lobby.PlayerReadyResponse_ErrorCode
		if err == lobby.ErrNotInRoom {
			errResponse = proto_lobby.PlayerReadyResponse_NOT_IN_ROOM.Enum()
		} else if err == lobby.ErrUnexpectedReady {
			errResponse = proto_lobby.PlayerReadyResponse_UNEXPECTED.Enum()
		} else if err == lobby.ErrInvalidStateString {
			errResponse = proto_lobby.PlayerReadyResponse_INVALID_STATE.Enum()
		} else if err != nil {
			log.Println("Unknown error: %s", err)
			return proto.CompositeMessage{Message: proto_errors.NewInternalErrorUnknown()}
		}
		response := proto_lobby.PlayerReadyResponse{
			ErrorCode: errResponse,
		}
		return proto.CompositeMessage{Message: &response}
	})
}

package service

import (
	"time"

	"code.google.com/p/go.net/context"

	"github.com/opentarock/service-api/go/client"
	"github.com/opentarock/service-api/go/proto"
	"github.com/opentarock/service-api/go/proto_errors"
	"github.com/opentarock/service-api/go/proto_headers"
	"github.com/opentarock/service-api/go/proto_lobby"
	"github.com/opentarock/service-api/go/reqcontext"
	"github.com/opentarock/service-api/go/service"
	"github.com/opentarock/service-api/go/user"
	"github.com/opentarock/service-lobby/lobby"
	"gopkg.in/inconshreveable/log15.v2"
)

const (
	defaultRequestTimeout = 10 * time.Second
	serviceName           = "lobby"
)

type lobbyServiceHandlers struct {
	roomList *lobby.RoomList
}

func NewLobbyServiceHandlers(notifyClient client.NotifyClient) *lobbyServiceHandlers {
	return &lobbyServiceHandlers{
		roomList: lobby.NewRoomList(notifyClient),
	}
}

func (s *lobbyServiceHandlers) CreateRoomHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.CreateRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		room, errCode := s.roomList.CreateRoom(user.Id(auth.GetUserId()), request.GetName(), request.GetOptions())
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
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.JoinRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		room, errCode := s.roomList.JoinRoom(user.Id(auth.GetUserId()), lobby.RoomId(request.GetRoomId()))

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
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.LeaveRoomRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		success, errCode := s.roomList.LeaveRoom(user.Id(auth.GetUserId()))

		response := proto_lobby.LeaveRoomResponse{}
		if !success {
			response.ErrorCode = errCode.Enum()
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) ListRoomsHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.ListRoomsRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		response := proto_lobby.ListRoomsResponse{
			Rooms: s.roomList.ListRoomsExcluding(user.Id(auth.GetUserId())),
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) RoomInfoHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.RoomInfoRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			logger.Error("Malformed request", "error", err)
			return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
		}
		response := proto_lobby.RoomInfoResponse{
			Room: s.roomList.GetRoom(lobby.RoomId(request.GetRoomId())),
		}
		if response.Room == nil {
			logger.Info("Room does not exist", "room_id", request.GetRoomId())
			response.ErrorCode = proto_lobby.RoomInfoResponse_ROOM_DOES_NOT_EXIST.Enum()
		} else {
			logger.Info("Getting room info", "room_id", request.GetRoomId())
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) StartGameHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.StartGameRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		err = s.roomList.StartGame(user.Id(auth.GetUserId()))
		var errResponse *proto_lobby.StartGameResponse_ErrorCode
		if err == lobby.ErrNotInRoom {
			errResponse = proto_lobby.StartGameResponse_NOT_IN_ROOM.Enum()
		} else if err == lobby.ErrNotOwner {
			errResponse = proto_lobby.StartGameResponse_NOT_OWNER.Enum()
		} else if err == lobby.ErrAlreadyStarted {
			errResponse = proto_lobby.StartGameResponse_ALREADY_STARTED.Enum()
		} else if err != nil {
			logger.Error("Unknown start game error", "error", err)
			return proto.CompositeMessage{Message: proto_errors.NewInternalErrorUnknown()}
		}
		response := proto_lobby.StartGameResponse{
			ErrorCode: errResponse,
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func (s *lobbyServiceHandlers) PlayerReadyHandler() service.MessageHandler {
	return service.MessageHandlerFunc(func(msg *proto.Message) proto.CompositeMessage {
		ctx, cancel := reqcontext.WithRequest(context.Background(), msg, defaultRequestTimeout)
		defer cancel()

		logger := reqcontext.ContextLogger(ctx, "service_name", serviceName)

		var request proto_lobby.PlayerReadyRequest
		err := msg.Unmarshal(&request)
		if err != nil {
			return newMalformedMessageError(logger, request.GetMessageType(), err)
		}

		auth, ok := reqcontext.AuthFromContext(ctx)
		if !ok {
			return missingAuthHeaderError(logger)
		}

		err = s.roomList.PlayerReady(user.Id(auth.GetUserId()), request.GetState())
		var errResponse *proto_lobby.PlayerReadyResponse_ErrorCode
		if err == lobby.ErrNotInRoom {
			errResponse = proto_lobby.PlayerReadyResponse_NOT_IN_ROOM.Enum()
		} else if err == lobby.ErrUnexpectedReady {
			errResponse = proto_lobby.PlayerReadyResponse_UNEXPECTED.Enum()
		} else if err == lobby.ErrInvalidStateString {
			errResponse = proto_lobby.PlayerReadyResponse_INVALID_STATE.Enum()
		} else if err != nil {
			logger.Error("Unknown player ready error", "error", err)
			return proto.CompositeMessage{Message: proto_errors.NewInternalErrorUnknown()}
		}
		response := proto_lobby.PlayerReadyResponse{
			ErrorCode: errResponse,
		}
		return proto.CompositeMessage{Message: &response}
	})
}

func newMalformedMessageError(logger log15.Logger, msgType proto.Type, err error) proto.CompositeMessage {
	logger.Error("Malformed request", "error", err, "msg_type", msgType)
	return proto.CompositeMessage{Message: proto_errors.NewMalformedMessageUnpack()}
}

func missingAuthHeaderError(logger log15.Logger) proto.CompositeMessage {
	headerType := proto_headers.AuthorizationHeaderMessage
	logger.Error("Missing authorization header", "msg_type", headerType)
	return proto.CompositeMessage{
		Message: proto_errors.NewMissingHeader(headerType),
	}
}

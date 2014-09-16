package lobby_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/opentarock/service-lobby/lobby"
)

const ownerId = uint64(1)

func makeRoom() *lobby.Room {
	return lobby.NewRoom("name", ownerId, 3)
}

func TestRoomIdIsGenerated(t *testing.T) {
	room1 := makeRoom()
	room2 := makeRoom()
	assert.NotEqual(t, room1.GetId(), room2.GetId())
}

func TestUserCanJoinRoom(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	assert.Equal(t, 2, room.NumPlayers())
	assert.Contains(t, room.GetUserIds(), uint64(2))
}

func TestUserCanNotJoinFullRoom(t *testing.T) {
	room := makeRoom()
	err := room.Join(2)
	assert.Nil(t, err)
	err = room.Join(3)
	assert.Nil(t, err)
	err = room.Join(4)
	assert.NotNil(t, err, "Error is returned if the room is full")
}

func TestUserCantJoinMultipleTimes(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Join(2)
	assert.Equal(t, 2, room.NumPlayers())
}

func TestUserCanLeaveRoom(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Join(3)
	assert.Equal(t, 3, room.NumPlayers())
	room.Leave(2)
	assert.Equal(t, 2, room.NumPlayers())
	room.Leave(3)
	assert.Equal(t, 1, room.NumPlayers())
}

func TestOwnerCanLeaveRoom(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Leave(ownerId)
	assert.Equal(t, 1, room.NumPlayers())
	assert.NotContains(t, room.GetUserIds(), uint64(ownerId))
	assert.Equal(t, 2, room.GetOwner(), "New owner is selected when the current owner leaves")
}

func TestReturnsUserIds(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	assert.Contains(t, room.GetUserIds(), uint64(2))
	assert.Contains(t, room.GetUserIds(), ownerId)

	room.Join(3)
	assert.Contains(t, room.GetUserIds(), uint64(3))
	assert.Contains(t, room.GetUserIds(), uint64(2))
	assert.Contains(t, room.GetUserIds(), ownerId)

	room.Leave(2)
	assert.Contains(t, room.GetUserIds(), uint64(3))
	assert.Contains(t, room.GetUserIds(), ownerId)
}

func TestLastUserCanLeave(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Join(3)
	assert.True(t, room.Leave(ownerId))
	assert.True(t, room.Leave(2))
	assert.False(t, room.Leave(3), "False is returned if there are no more users in the room")
	assert.Equal(t, 0, room.GetOwner())
}

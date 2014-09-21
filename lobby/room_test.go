package lobby_test

import (
	"testing"
	"time"

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

func TestReturnsNonOwnerUserIds(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	assert.Equal(t, 1, len(room.GetNonOwnerUserIds()))
	assert.Contains(t, room.GetNonOwnerUserIds(), uint64(2))
	assert.NotContains(t, room.GetNonOwnerUserIds(), ownerId)

	room.Leave(2)
	assert.NotContains(t, room.GetNonOwnerUserIds(), uint64(2))
	assert.NotContains(t, room.GetNonOwnerUserIds(), ownerId)
}

func TestLastUserCanLeave(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Join(3)

	nonEmpty, err := room.Leave(ownerId)
	assert.Nil(t, err)
	assert.True(t, nonEmpty)

	nonEmpty, err = room.Leave(2)
	assert.Nil(t, err)
	assert.True(t, nonEmpty)

	nonEmpty, err = room.Leave(2)
	assert.Nil(t, err)
	assert.False(t, nonEmpty, "False is returned if there are no more users in the room")
	assert.Equal(t, 0, room.GetOwner())
}

func TestGameStartCanBeCancelled(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	assert.False(t, room.IsStarting())
	room.StartGame()
	assert.True(t, room.IsStarting())
	err := room.CancelStart()
	assert.Nil(t, err)
	assert.False(t, room.IsStarting())
}

func TestRoomGameCantBeStartedMultipleTimes(t *testing.T) {
	room := makeRoom()
	_, err := room.StartGame()
	assert.Nil(t, err)
	defer room.CancelStart()
	_, err = room.StartGame()
	assert.Equal(t, lobby.ErrAlreadyStarted, err)
}

func TestNotStartingStatusGameCantBeCancelled(t *testing.T) {
	room := makeRoom()
	err := room.CancelStart()
	assert.Equal(t, lobby.ErrNotStarting, err)
}

func TestUserCantLeaveWhenGameStartInProgress(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.StartGame()
	defer room.CancelStart()

	_, err := room.Leave(2)
	assert.Equal(t, lobby.ErrGameStartInProgress, err)
}

func TestUserCanjoinWhileGameStartInProgressAndLeave(t *testing.T) {
	room := makeRoom()
	room.StartGame()
	defer room.CancelStart()

	err := room.Join(2)
	assert.Nil(t, err)
	nonEmpty, err := room.Leave(2)
	assert.True(t, nonEmpty)
	assert.Nil(t, err)
}

func TestOwnerCantLeaveWhenGameStartInProgress(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.StartGame()
	defer room.CancelStart()

	_, err := room.Leave(ownerId)
	assert.Equal(t, lobby.ErrGameStartInProgress, err)
}

func TestGameIsStartedWhenAllPlayersAreReady(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	room.Join(3)
	playerState, _ := room.StartGame()
	assert.True(t, room.IsStarting(), "After start room status should be starting")
	room.PlayerReady(2, playerState[2])
	assert.True(t, room.IsStarting(), "Until the room is full room status should be starting")
	room.PlayerReady(3, playerState[3])
	assert.True(t, room.IsInProgress(), "After all the players are ready game should be started")
}

func TestGameStatusIsResetAfterPlayerReadyTImeout(t *testing.T) {
	room := makeRoom()
	room.ReadyTimeout = 100 * time.Millisecond
	room.Join(2)
	room.StartGame()
	assert.True(t, room.IsStarting(), "After start room status should be starting")
	time.Sleep(200 * time.Millisecond)
	assert.True(t, !room.IsStarted(), "After timeout room status should be back to notStarted")
}

func TestUserStateIsRegeneratedOnTimeout(t *testing.T) {
	room := makeRoom()
	room.ReadyTimeout = 100 * time.Millisecond
	room.Join(2)
	playerState, _ := room.StartGame()
	time.Sleep(200 * time.Millisecond)
	playerState2, _ := room.StartGame()
	defer room.CancelStart()
	assert.NotEqual(t, playerState[2], playerState2[2], "State should be regenerated")
}

func TestPlayerCantReadyIfGameIsNotStarting(t *testing.T) {
	room := makeRoom()
	room.Join(2)
	err := room.PlayerReady(2, "state")
	assert.Equal(t, lobby.ErrUnexpectedReady, err)
}

func TestGameWithOnlyOnePLayerIsAutomaticallyStarted(t *testing.T) {
	room := makeRoom()
	playerState, err := room.StartGame()
	assert.Nil(t, err)
	assert.Equal(t, len(playerState), 0)
	assert.True(t, room.IsInProgress(), "Game should be automatically started")
}

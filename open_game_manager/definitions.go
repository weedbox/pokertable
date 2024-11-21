package open_game_manager

import (
	"errors"

	"github.com/weedbox/syncsaga"
)

var (
	ErrParticipantNotFound = errors.New("open_game_manager: participant not found")
)

type OpenGameManager interface {
	Ready(participantID string) error
	Setup(gameCount int, participants map[string]int)
	GetState() OpenGameState
	PrintState()
}

type openGameManager struct {
	onOpenGameReady func(state OpenGameState)
	rg              *syncsaga.ReadyGroup
	state           *OpenGameState
}

type OpenGameOption struct {
	Timeout         int
	OnOpenGameReady func(state OpenGameState)
}

type OpenGameState struct {
	Timeout      int                             `json:"timeout"`
	GameCount    int                             `json:"game_count"`
	Participants map[string]*OpenGameParticipant `json:"participants"` // key: participant_id, value: participant
}

type OpenGameParticipant struct {
	ID      string `json:"id"`
	Index   int    `json:"index"`
	IsReady bool   `json:"is_ready"`
}

package open_game_manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_InitOpenGameManager(t *testing.T) {
	options := OpenGameOption{
		Timeout: 1,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("OpenGameReady for game count: ", state.GameCount)
		},
	}

	m := NewOpenGameManager(options)

	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))
}

func Test_InitOpenGameManagerFromState(t *testing.T) {
	fmt.Println("[Test_InitOpenGameManagerFromState] Start: ", time.Now().Format(time.RFC3339))
	options := OpenGameOption{
		Timeout: 1,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.Equal(t, true, participant.IsReady)
			}
			fmt.Println("[Test_InitOpenGameManagerFromState] Done: ", time.Now().Format(time.RFC3339))
		},
	}
	state := OpenGameState{
		Timeout:   options.Timeout,
		GameCount: 5,
		Participants: map[string]*OpenGameParticipant{
			"player 1": {
				ID:      "player 1",
				Index:   1,
				IsReady: true,
			},
			"player 2": {
				ID:      "player 2",
				Index:   2,
				IsReady: false,
			},
			"player 3": {
				ID:      "player 3",
				Index:   3,
				IsReady: true,
			},
		},
	}
	m := NewOpenGameManagerFromState(state, options)
	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, state.GameCount, m.GetState().GameCount)
	assert.Equal(t, len(state.Participants), len(m.GetState().Participants))
	for _, participant := range state.Participants {
		assert.Equal(t, m.GetState().Participants[participant.ID].ID, participant.ID)
		assert.Equal(t, m.GetState().Participants[participant.ID].Index, participant.Index)
		assert.Equal(t, m.GetState().Participants[participant.ID].IsReady, participant.IsReady)
	}

	time.Sleep(3 * time.Second)
	fmt.Println("[Test_InitOpenGameManagerFromState] End: ", time.Now().Format(time.RFC3339))
	// m.PrintState()
}

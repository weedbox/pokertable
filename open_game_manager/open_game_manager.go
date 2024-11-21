package open_game_manager

import (
	"encoding/json"
	"fmt"

	"github.com/weedbox/syncsaga"
)

func NewOpenGameManager(options OpenGameOption) OpenGameManager {
	m := &openGameManager{
		onOpenGameReady: options.OnOpenGameReady,
		rg: syncsaga.NewReadyGroup(syncsaga.WithTimeout(options.Timeout, func(rg *syncsaga.ReadyGroup) {
			// Auto Ready By Default
			for idx, isReady := range rg.GetParticipantStates() {
				if !isReady {
					rg.Ready(idx)
				}
			}
		})),
	}
	m.state = &OpenGameState{
		Timeout:      options.Timeout,
		GameCount:    0,
		Participants: make(map[string]*OpenGameParticipant),
	}

	return m
}

func NewOpenGameManagerFromState(state OpenGameState, options OpenGameOption) OpenGameManager {
	m := &openGameManager{
		onOpenGameReady: options.OnOpenGameReady,
		rg: syncsaga.NewReadyGroup(syncsaga.WithTimeout(options.Timeout, func(rg *syncsaga.ReadyGroup) {
			// Auto Ready By Default
			for idx, isReady := range rg.GetParticipantStates() {
				if !isReady {
					rg.Ready(idx)
				}
			}
		})),
		state: &OpenGameState{
			Timeout:      options.Timeout,
			GameCount:    state.GameCount,
			Participants: make(map[string]*OpenGameParticipant),
		},
	}
	m.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		m.readyGroupOnCompleted()
	})

	m.readyGroupResetParticipants()
	readyParticipants := make(map[int]OpenGameParticipant, 0)
	for _, participant := range state.Participants {
		m.readyGroupAddParticipant(*participant, false)
		if participant.IsReady {
			readyParticipants[participant.Index] = OpenGameParticipant{
				ID:      participant.ID,
				Index:   participant.Index,
				IsReady: true,
			}
		}
	}
	m.rg.Start()

	for _, readyParticipant := range readyParticipants {
		m.readyGroupAddParticipant(readyParticipant, true)
	}

	return m
}

func (m *openGameManager) Ready(participantID string) error {
	return m.readyGroupReady(participantID)
}

func (m *openGameManager) Setup(gameCount int, participants map[string]int) {
	m.state.GameCount = gameCount

	m.rg.Stop()
	m.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		m.readyGroupOnCompleted()
	})
	m.readyGroupResetParticipants()
	for id, idx := range participants {
		participant := OpenGameParticipant{
			ID:      id,
			Index:   idx,
			IsReady: false,
		}
		m.readyGroupAddParticipant(participant, false)
	}

	m.rg.Start()
}

func (m *openGameManager) GetState() OpenGameState {
	return *m.state
}

func (m *openGameManager) PrintState() {
	encoded, err := json.Marshal(m.state)
	if err != nil {
		fmt.Println("state: nil")
	} else {
		fmt.Println("state:", string(encoded))
	}
}

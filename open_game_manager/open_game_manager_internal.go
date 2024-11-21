package open_game_manager

func (m *openGameManager) readyGroupResetParticipants() {
	m.rg.ResetParticipants()
	m.state.Participants = map[string]*OpenGameParticipant{}
}

func (m *openGameManager) readyGroupAddParticipant(participant OpenGameParticipant, isReady bool) {
	m.state.Participants[participant.ID] = &OpenGameParticipant{
		ID:      participant.ID,
		Index:   participant.Index,
		IsReady: isReady,
	}
	m.rg.Add(int64(participant.Index), isReady)
}

func (m *openGameManager) readyGroupOnCompleted() {
	for participantID := range m.state.Participants {
		m.state.Participants[participantID].IsReady = true
	}
	m.onOpenGameReady(m.GetState())
}

func (m *openGameManager) readyGroupReady(participantID string) error {
	participant, exist := m.state.Participants[participantID]
	if !exist {
		return ErrParticipantNotFound
	}

	m.rg.Ready(int64(participant.Index))
	participant.IsReady = true
	return nil
}

package open_game_manager

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
)

func TestOpenGameManager_Init(t *testing.T) {
	options := OpenGameOption{
		Timeout: 1,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_Init] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_Init] Done: ", time.Now().Format(time.RFC3339))
		},
	}

	m := NewOpenGameManager(options)

	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))
}

func TestOpenGameManager_InitFromState(t *testing.T) {
	fmt.Println("[TestOpenGameManager_InitFromState] Start: ", time.Now().Format(time.RFC3339))
	options := OpenGameOption{
		Timeout: 1,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_InitFromState] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_InitFromState] Done: ", time.Now().Format(time.RFC3339))
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

func TestOpenGameManager_Setup(t *testing.T) {
	fmt.Println("[TestOpenGameManager_Setup] Start: ", time.Now().Format(time.RFC3339))
	options := OpenGameOption{
		Timeout: 5,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_Setup] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_Setup] Done: ", time.Now().Format(time.RFC3339))
		},
	}
	participants := map[string]int{
		"player 1": 1,
		"player 2": 2,
		"player 3": 3,
	}
	gameCount := 10

	m := NewOpenGameManager(options)

	// 檢查初始化狀態
	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))

	m.Setup(gameCount, participants)

	// 驗證 Setup 後狀態
	state := m.GetState()
	assert.Equal(t, gameCount, state.GameCount)
	assert.Equal(t, len(participants), len(state.Participants))
	for id, index := range participants {
		participant, exists := state.Participants[id]
		assert.True(t, exists)
		assert.Equal(t, index, participant.Index)
		assert.Equal(t, id, participant.ID)
		assert.False(t, state.Participants[id].IsReady)
	}

	fmt.Println("[TestOpenGameManager_Setup] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupAllReady(t *testing.T) {
	fmt.Println("[TestOpenGameManager_SetupAllReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 3, // 超時時間為 3 秒
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_SetupAllReady] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_SetupAllReady] Done: ", time.Now().Format(time.RFC3339))
		},
	}
	gameCount := 3
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}

	m := NewOpenGameManager(options)

	// 檢查初始化狀態
	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))

	m.Setup(gameCount, participants)

	// 驗證 Setup 後狀態
	state := m.GetState()
	assert.Equal(t, gameCount, state.GameCount)
	assert.Equal(t, len(participants), len(state.Participants))
	for id, index := range participants {
		participant, exists := state.Participants[id]
		assert.True(t, exists)
		assert.Equal(t, index, participant.Index)
		assert.Equal(t, id, participant.ID)
		assert.False(t, state.Participants[id].IsReady)
	}

	// 逐一設定參與者為就緒
	for _, participant := range state.Participants {
		go func(participant OpenGameParticipant) {
			time.Sleep(time.Second * 1)
			assert.NoError(t, m.Ready(participant.ID))
		}(*participant)

	}

	time.Sleep(3 * time.Second)

	// 等待超時後驗證狀態
	for _, participant := range m.GetState().Participants {
		assert.True(t, participant.IsReady) // 超時後所有參與者應變為就緒
	}

	fmt.Println("[TestOpenGameManager_SetupAllReady] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupAllNotReady(t *testing.T) {
	fmt.Println("[TestOpenGameManager_SetupNotAllReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 2, // 超時時間為 2 秒
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_SetupNotAllReady] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_SetupNotAllReady] Done: ", time.Now().Format(time.RFC3339))
		},
	}
	gameCount := 3
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}

	m := NewOpenGameManager(options)

	// 檢查初始化狀態
	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))

	m.Setup(gameCount, participants)

	// 驗證 Setup 後狀態
	state := m.GetState()
	assert.Equal(t, gameCount, state.GameCount)
	assert.Equal(t, len(participants), len(state.Participants))
	for id, index := range participants {
		participant, exists := state.Participants[id]
		assert.True(t, exists)
		assert.Equal(t, index, participant.Index)
		assert.Equal(t, id, participant.ID)
		assert.False(t, state.Participants[id].IsReady)
	}

	time.Sleep(5 * time.Second)

	// 等待超時後驗證狀態
	for _, participant := range m.GetState().Participants {
		assert.True(t, participant.IsReady) // 超時後所有參與者應變為就緒
	}

	fmt.Println("[TestOpenGameManager_SetupNotAllReady] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupPartialReady(t *testing.T) {
	fmt.Println("[TestOpenGameManager_SetupPartialReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 2, // 超時時間為 2 秒
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("[TestOpenGameManager_SetupPartialReady] OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.Equal(t, state.Participants[participant.ID].ID, participant.ID)
				assert.Equal(t, state.Participants[participant.ID].Index, participant.Index)
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_SetupPartialReady] Done: ", time.Now().Format(time.RFC3339))
		},
	}
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}
	excludePlayerIDs := []string{"player1", "player2"}
	gameCount := 1

	m := NewOpenGameManager(options)

	// 檢查初始化狀態
	assert.Equal(t, options.Timeout, m.GetState().Timeout)
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))

	m.Setup(gameCount, participants)

	// 驗證 Setup 後狀態
	state := m.GetState()
	assert.Equal(t, gameCount, state.GameCount)
	assert.Equal(t, len(participants), len(state.Participants))
	for id, index := range participants {
		participant, exists := state.Participants[id]
		assert.True(t, exists)
		assert.Equal(t, index, participant.Index)
		assert.Equal(t, id, participant.ID)
		assert.False(t, state.Participants[id].IsReady)
	}

	// 設定部分參與者為就緒
	for playerID := range participants {
		if !funk.Contains(excludePlayerIDs, playerID) {
			go func(playerID string) {
				time.Sleep(500 * time.Microsecond)
				assert.NoError(t, m.Ready(playerID))
			}(playerID)
		}
	}

	time.Sleep(8 * time.Second)

	// 等待超時後驗證狀態
	for _, participant := range m.GetState().Participants {
		assert.True(t, participant.IsReady) // 超時後所有參與者應變為就緒
	}

	fmt.Println("[TestOpenGameManager_SetupPartialReady] End: ", time.Now().Format(time.RFC3339))
}

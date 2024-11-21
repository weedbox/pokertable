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
			fmt.Println("OpenGameReady for game count: ", state.GameCount)
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
			fmt.Println("OpenGameReady for game count: ", state.GameCount)
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
		Timeout: 2,
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("OpenGameReady for game count: ", state.GameCount)
			for _, participant := range state.Participants {
				assert.True(t, participant.IsReady)
			}
		},
	}
	newParticipants := map[string]int{
		"player 1": 1,
		"player 2": 2,
		"player 3": 3,
	}
	gameCount := 10

	m := NewOpenGameManager(options)

	// 檢查初始化狀態
	assert.Equal(t, 0, m.GetState().GameCount)
	assert.Equal(t, 0, len(m.GetState().Participants))

	m.Setup(gameCount, newParticipants)
	// 驗證 Setup 後狀態
	state := m.GetState()

	assert.Equal(t, gameCount, state.GameCount)
	assert.Equal(t, len(newParticipants), len(state.Participants))
	for id, index := range newParticipants {
		participant, exists := state.Participants[id]
		assert.True(t, exists)
		assert.Equal(t, index, participant.Index)
		assert.Equal(t, id, participant.ID)
		assert.False(t, state.Participants[id].IsReady)
	}

	for _, participant := range state.Participants {
		go func(participant OpenGameParticipant) {
			time.Sleep(time.Duration(participant.Index*100) * time.Microsecond)

			assert.NoError(t, m.Ready(participant.ID))
			assert.True(t, m.GetState().Participants[participant.ID].IsReady)
		}(*participant)
	}

	time.Sleep(3 * time.Second)

	fmt.Println("[TestOpenGameManager_Setup] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupAllReady(t *testing.T) {
	fmt.Println("[TestOpenGameManager_SetupAllReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 5, // 超時時間為 3 秒
		OnOpenGameReady: func(state OpenGameState) {
			fmt.Println("OnOpenGameReady triggered for game count:", state.GameCount)
			for _, participant := range state.Participants {
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_SetupAllReady] All participants are ready.")
		},
	}
	gameCount := 3
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}

	m := NewOpenGameManager(options)

	m.Setup(gameCount, participants)
	state := m.GetState()
	// 驗證初始狀態
	for _, participant := range state.Participants {
		fmt.Println(participant.ID, "initially ready:", participant.IsReady)
		assert.False(t, participant.IsReady) // 初始狀態應為就緒
	}

	// 驗證 Setup 後狀態
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
		fmt.Println(participant.ID, "ready after timeout:", participant.IsReady)
	}

	fmt.Println("[TestOpenGameManager_SetupAllReady] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupNotAllReady(t *testing.T) {
	fmt.Println("[Test_Setup_AllReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 2, // 超時時間為 2 秒
		OnOpenGameReady: func(state OpenGameState) {
			for _, participant := range state.Participants {
				assert.True(t, participant.IsReady)
			}
			fmt.Println("[TestOpenGameManager_SetupNotAllReady] All participants are ready.")
		},
	}
	// 使用 Setup 方法重置狀態
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}

	m := NewOpenGameManager(options)

	// 驗證初始狀態
	for _, participant := range m.GetState().Participants {
		assert.False(t, participant.IsReady) // 初始狀態應為就緒
	}

	m.Setup(3, participants)

	state := m.GetState()
	// 驗證 Setup 後狀態
	for _, participant := range state.Participants {
		assert.False(t, participant.IsReady) // Setup 應將所有參與者狀態重置為未就緒
	}

	time.Sleep(5 * time.Second)

	fmt.Println("[TestOpenGameManager_SetupNotAllReady] End: ", time.Now().Format(time.RFC3339))
}

func TestOpenGameManager_SetupPartialReady(t *testing.T) {
	fmt.Println("[TestOpenGameManager_SetupPartialReady] Start: ", time.Now().Format(time.RFC3339))

	options := OpenGameOption{
		Timeout: 2, // 超時時間為 2 秒
		OnOpenGameReady: func(state OpenGameState) {
			for _, participant := range state.Participants {
				assert.True(t, participant.IsReady)
			}
		},
	}
	// 使用 Setup 方法重置狀態
	participants := map[string]int{
		"player1": 1,
		"player2": 2,
		"player3": 3,
	}
	excludes := []string{"player1", "player2"}
	gameCount := 1

	m := NewOpenGameManager(options)
	// 驗證初始狀態
	for key := range participants {
		if !funk.Contains(key, excludes) {
			go func(key string) {
				time.Sleep(500 * time.Microsecond)
				m.Ready(key)
			}(key)
		}
	}

	m.Setup(gameCount, participants)

	// 驗證 Setup 後狀態
	for key, participant := range m.GetState().Participants {
		assert.Equal(t, funk.Contains(key, excludes), participant.IsReady)
	}
	time.Sleep(3 * time.Second)

	// 等待超時後驗證狀態
	fmt.Println("[TestOpenGameManager_SetupPartialReady] End: ", time.Now().Format(time.RFC3339))
}

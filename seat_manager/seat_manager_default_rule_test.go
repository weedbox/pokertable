package seat_manager

import (
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
)

func TestDefaultRule_InitSeatManager(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default

	sm := NewSeatManager(maxSeat, rule)

	assert.Equal(t, UnsetSeatID, sm.CurrentDealerSeatID())
	assert.Equal(t, UnsetSeatID, sm.CurrentSBSeatID())
	assert.Equal(t, UnsetSeatID, sm.CurrentBBSeatID())
	assert.False(t, sm.IsInitPositions())
	for _, seatPlayer := range sm.Seats() {
		assert.Nil(t, seatPlayer)
	}
}

func TestDefaultRule_BatchRandomAssignSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)

	assert.NoError(t, err)

	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.NotEqual(t, UnsetSeatID, seatID)
			assert.Greater(t, seatID, UnsetSeatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// P1 & P3 join in the seats
	joinPlayerIDs := []string{"P1", "P3"}
	err = sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}
}

func TestDefaultRule_ParallelRandomAssignSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}
	var wg sync.WaitGroup

	sm := NewSeatManager(maxSeat, rule)

	for _, playerID := range playerIDs {
		wg.Add(1)
		go func(playerID string) {
			defer wg.Done()
			err := sm.RandomAssignSeats([]string{playerID})
			assert.NoError(t, err)
		}(playerID)
	}
	wg.Wait()

	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.NotEqual(t, UnsetSeatID, seatID)
			assert.Greater(t, seatID, UnsetSeatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// P1 & P3 join in the seats
	joinPlayerIDs := []string{"P1", "P3"}
	err := sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}
}

func TestDefaultRule_SyncRandomAssignSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)

	for _, playerID := range playerIDs {
		err := sm.RandomAssignSeats([]string{playerID})
		assert.NoError(t, err)
	}

	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.NotEqual(t, UnsetSeatID, seatID)
			assert.Greater(t, seatID, UnsetSeatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// P1 & P3 join in the seats
	joinPlayerIDs := []string{"P1", "P3"}
	err := sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}
}

func TestDefaultRule_RandomAssignSeats_ErrNotEnoughSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)

	assert.ErrorIs(t, err, ErrNotEnoughSeats)
}

func TestDefaultRule_AssignSeats_BeforeInitPositions(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P4": 5,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)

	assert.NoError(t, err)

	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedSeatID, exist := playerSeatIDs[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedSeatID, seatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// All players join in the seats
	joinPlayerIDs := []string{"P1", "P2", "P4"}
	err = sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}
}

func TestDefaultRule_AssignSeats_BeforeInitPositions_MultipleTimes(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 5,
	}
	newPlayerSeatIDs := map[string]int{
		"P4": 1,
		"P5": 7,
		"P6": 8,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)
	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedSeatID, exist := playerSeatIDs[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedSeatID, seatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	allPlayerSeatIDs := playerSeatIDs
	maps.Copy(allPlayerSeatIDs, newPlayerSeatIDs)
	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedSeatID, exist := allPlayerSeatIDs[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedSeatID, seatID)
			assert.NotEqual(t, UnsetSeatID, seatID)
			assert.Greater(t, seatID, UnsetSeatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// P1 & P3 join in the seats
	joinPlayerIDs := []string{"P1", "P3"}
	err = sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}
}

func TestDefaultRule_AssignSeats_AfterInitPositions_MultipleTimes(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 5,
	}
	newPlayerSeatIDs := map[string]int{
		"P4": 1,
		"P5": 7,
		"P6": 8,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)
	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedSeatID, exist := playerSeatIDs[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedSeatID, seatID)
			assert.False(t, seatPlayer.Active())
		}
	}

	// P1, P2 & P3 join in the seats
	joinPlayerIDs := []string{"P1", "P2", "P3"}
	err = sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			if funk.Contains(joinPlayerIDs, seatPlayer.ID) {
				assert.True(t, seatPlayer.Active())
			} else {
				assert.False(t, seatPlayer.Active())
			}
		}
	}

	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	DebugPrintSeats("init", sm)

	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// P4, P5 & P6 join in the seats
	joinPlayerIDs = []string{"P4", "P5", "P6"}
	err = sm.JoinPlayers(joinPlayerIDs)
	assert.NoError(t, err)

	DebugPrintSeats("add new players", sm)

	expectedPlayerActiveStates := map[string]bool{
		"P1": true, // bb
		"P2": true, // dealer
		"P3": true, // sb
		"P4": true,
		"P5": false,
		"P6": false,
	}

	allPlayerSeatIDs := playerSeatIDs
	maps.Copy(allPlayerSeatIDs, newPlayerSeatIDs)
	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedSeatID, exist := allPlayerSeatIDs[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedSeatID, seatID)
			assert.Equal(t, expectedPlayerActiveStates[seatPlayer.ID], seatPlayer.Active())
		}
	}
}

func TestDefaultRule_AssignSeats_ErrNotEnoughSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1":  0,
		"P2":  1,
		"P3":  2,
		"P4":  3,
		"P5":  4,
		"P6":  5,
		"P7":  6,
		"P8":  7,
		"P9":  8,
		"P10": 9,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)

	assert.ErrorIs(t, err, ErrNotEnoughSeats)
}

func TestDefaultRule_AssignSeats_ErrSeatAlreadyIsTaken(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	newPlayerSeatIDs := map[string]int{
		"P3": 0,
	}

	sm := NewSeatManager(maxSeat, rule)

	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.ErrorIs(t, err, ErrSeatAlreadyIsTaken)
}

func TestDefaultRule_AssignSeats_ErrDuplicateSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 0,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)

	assert.ErrorIs(t, err, ErrDuplicateSeats)
}

func TestDefaultRule_AssignSeats_ErrDuplicatePlayers2(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	newPlayerSeatIDs := map[string]int{
		"P1": 5,
	}

	sm := NewSeatManager(maxSeat, rule)

	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.ErrorIs(t, err, ErrDuplicatePlayers)
}

func TestDefaultRule_GetSeat(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P4": 5,
	}

	sm := NewSeatManager(maxSeat, rule)

	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	for playerID, expectedSeatID := range playerSeatIDs {
		seatID, err := sm.GetSeatID(playerID)
		assert.NoError(t, err)
		assert.Equal(t, expectedSeatID, seatID)
	}
}

func TestDefaultRule_GetSeat_ErrPlayerNotFound(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P4": 5,
	}

	sm := NewSeatManager(maxSeat, rule)

	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	seatID, err := sm.GetSeatID("P10")
	assert.ErrorIs(t, err, ErrPlayerNotFound)
	assert.Equal(t, UnsetSeatID, seatID)
}

func TestDefaultRule_RemoveSeats(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	removedPlayers := []string{"P3", "P5"}
	err = sm.RemoveSeats(removedPlayers)

	assert.NoError(t, err)
}

func TestDefaultRule_RemoveSeats_ErrPlayerNotFound(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	removedPlayers := []string{"P30", "P5"}
	err = sm.RemoveSeats(removedPlayers)
	assert.ErrorIs(t, err, ErrPlayerNotFound)
}

func TestDefaultRule_UpdateSeatPlayerActiveStates(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	expectedPlayerActiveStates := map[string]bool{
		"P1": true,
		"P2": true,
		"P3": false,
		"P4": true,
		"P5": false,
	}

	joinPlayers := []string{"P1", "P2", "P4"}
	err = sm.JoinPlayers(joinPlayers)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			expectedActiveState, exist := expectedPlayerActiveStates[seatPlayer.ID]
			assert.True(t, exist)
			assert.Equal(t, expectedActiveState, seatPlayer.Active())
		}
	}
}

func TestDefaultRule_UpdateSeatPlayerActiveStates_ErrPlayerNotFound(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	joinPlayers := []string{"P1", "P2", "P3", "P4", "P5", "P10"}
	err = sm.JoinPlayers(joinPlayers)
	assert.ErrorIs(t, err, ErrPlayerNotFound)
}

func TestDefaultRule_RandomInitPositions_TwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	// P1 & P2 join
	joinPlayers := []string{"P1", "P2"}
	err = sm.JoinPlayers(joinPlayers)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, joinPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(true)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	assert.NotEqual(t, UnsetSeatID, sm.CurrentDealerSeatID())
	assert.NotEqual(t, UnsetSeatID, sm.CurrentSBSeatID())
	assert.NotEqual(t, UnsetSeatID, sm.CurrentBBSeatID())
	assert.Equal(t, sm.CurrentDealerSeatID(), sm.CurrentSBSeatID())
	assert.NotEqual(t, sm.CurrentBBSeatID(), sm.CurrentDealerSeatID())
}

func TestDefaultRule_NotRandomInitPositions_TwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0, // bb
		"P2": 3, // dealer/sb
	}
	expectedSeatPositions := map[string]int{
		Position_Dealer: 3,
		Position_SB:     3,
		Position_BB:     0,
	}
	expectedPlayerPositions := map[string][]string{
		"P1": {Position_BB},
		"P2": {Position_Dealer, Position_SB},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// P1 & P2 join
	joinPlayers := []string{"P1", "P2"}
	err = sm.JoinPlayers(joinPlayers)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, joinPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_RandomInitPositions_MoreThanTwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	// join all players
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(true)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	assert.NotEqual(t, UnsetSeatID, sm.CurrentDealerSeatID())
	assert.NotEqual(t, UnsetSeatID, sm.CurrentSBSeatID())
	assert.NotEqual(t, UnsetSeatID, sm.CurrentBBSeatID())
	assert.NotContains(t, []int{sm.CurrentBBSeatID(), sm.CurrentSBSeatID()}, sm.CurrentDealerSeatID())
	assert.NotContains(t, []int{sm.CurrentBBSeatID(), sm.CurrentDealerSeatID()}, sm.CurrentSBSeatID())
	assert.NotContains(t, []int{sm.CurrentDealerSeatID(), sm.CurrentSBSeatID()}, sm.CurrentBBSeatID())
}

func TestDefaultRule_NotRandomInitPositions_MoreThanTwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0, // bb
		"P2": 3, // ug
		"P3": 4, // dealer
		"P4": 7, // sb
	}
	expectedSeatPositions := map[string]int{
		Position_Dealer: 4,
		Position_SB:     7,
		Position_BB:     0,
	}
	expectedPlayerPositions := map[string][]string{
		"P1": {Position_BB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_SB},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_InitPositions_ErrUnableToInitPositions_InvalidRule(t *testing.T) {
	maxSeat := 9
	rule := "InvalidRule"
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	err = sm.InitPositions(true)
	assert.ErrorIs(t, err, ErrUnableToInitPositions)
}

func TestDefaultRule_InitPositions_ErrAlreadyInitPositions_DuplicateInit(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1", "P2", "P3", "P4", "P5"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	// join all players
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(true)
	assert.NoError(t, err)

	err = sm.InitPositions(true)
	assert.ErrorIs(t, err, ErrAlreadyInitPositions)
}

func TestDefaultRule_InitPositions_ErrUnableToInitPositions_SinglePlayer(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerIDs := []string{"P1"}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.RandomAssignSeats(playerIDs)
	assert.NoError(t, err)

	// join all players
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	err = sm.InitPositions(true)
	assert.ErrorIs(t, err, ErrUnableToInitPositions)
}

func TestDefaultRule_RotatePositions_MultipleTimes_TwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	expectedSeatPositions_OddGameCounts := map[string]int{
		Position_Dealer: 3,
		Position_SB:     3,
		Position_BB:     0,
	}
	expectedPlayerPositions_OddGameCounts := map[string][]string{
		"P1": {Position_BB},
		"P2": {Position_Dealer, Position_SB},
	}
	expectedSeatPositions_EvenGameCounts := map[string]int{
		Position_Dealer: 0,
		Position_SB:     0,
		Position_BB:     3,
	}
	expectedPlayerPositions_EvenGameCounts := map[string][]string{
		"P1": {Position_Dealer, Position_SB},
		"P2": {Position_BB},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	for gameCount := 1; gameCount <= 10; gameCount++ {
		var err error
		if gameCount == 1 {
			err = sm.InitPositions(false)
			assert.NoError(t, err)
			assert.True(t, sm.IsInitPositions())

			verifySeatsAndPlayerPositions(t, expectedSeatPositions_OddGameCounts, expectedPlayerPositions_OddGameCounts, sm)
		} else {
			err = sm.RotatePositions()
			assert.NoError(t, err)

			if gameCount%2 == 0 {
				verifySeatsAndPlayerPositions(t, expectedSeatPositions_EvenGameCounts, expectedPlayerPositions_EvenGameCounts, sm)
			} else {
				verifySeatsAndPlayerPositions(t, expectedSeatPositions_OddGameCounts, expectedPlayerPositions_OddGameCounts, sm)
			}
		}
	}
}

func TestDefaultRule_RotatePositions_HU_NextRound_AddPlayersBetweenDealerBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	allPlayers := make([]string, 0)
	gameCount1PlayerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	expectedSeatPositions_GameCount1 := map[string]int{
		Position_Dealer: 3,
		Position_SB:     3,
		Position_BB:     0,
	}
	expectedPlayerPositions_GameCount1 := map[string][]string{
		"P1": {Position_BB},
		"P2": {Position_Dealer, Position_SB},
	}

	gameCount2PlayerSeatIDs := map[string]int{
		"P3": 6,
		"P4": 4,
	}
	expectedSeatPositions_GameCount2 := map[string]int{
		Position_Dealer: 6,
		Position_SB:     0,
		Position_BB:     3,
	}
	expectedPlayerPositions_GameCount2 := map[string][]string{
		"P1": {Position_SB},
		"P2": {Position_BB},
		"P3": {Position_Dealer},
		"P4": {},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(gameCount1PlayerSeatIDs)
	assert.NoError(t, err)

	// join all players
	gameCount1PlayerIDs := make([]string, 0)
	for playerID := range gameCount1PlayerSeatIDs {
		gameCount1PlayerIDs = append(gameCount1PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount1PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount1PlayerIDs...)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// GameCount = 1, P1 & P2 players are playing
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount1, expectedPlayerPositions_GameCount1, sm)

	// GameCount = 2, add P3
	err = sm.AssignSeats(gameCount2PlayerSeatIDs)
	assert.NoError(t, err)

	// join P3
	gameCount2PlayerIDs := make([]string, 0)
	for playerID := range gameCount2PlayerSeatIDs {
		gameCount2PlayerIDs = append(gameCount2PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount2PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount2PlayerIDs...)

	err = sm.RotatePositions()
	assert.NoError(t, err)

	DebugPrintSeats("GameCount = 2", sm)

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount2, expectedPlayerPositions_GameCount2, sm)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}
}

func TestDefaultRule_RotatePositions_HU_NextRound_AddPlayersNotBetweenDealerBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	allPlayers := make([]string, 0)
	gameCount1PlayerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	expectedSeatPositions_GameCount1 := map[string]int{
		Position_Dealer: 3,
		Position_SB:     3,
		Position_BB:     0,
	}
	expectedPlayerPositions_GameCount1 := map[string][]string{
		"P1": {Position_BB},
		"P2": {Position_Dealer, Position_SB},
	}

	gameCount2PlayerSeatIDs := map[string]int{
		"P3": 2,
		"P4": 1,
	}
	expectedSeatPositions_GameCount2 := map[string]int{
		Position_Dealer: 3,
		Position_SB:     0,
		Position_BB:     1,
	}
	expectedPlayerPositions_GameCount2 := map[string][]string{
		"P1": {Position_SB},
		"P2": {Position_Dealer},
		"P3": {},
		"P4": {Position_BB},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(gameCount1PlayerSeatIDs)
	assert.NoError(t, err)

	// join all players
	gameCount1PlayerIDs := make([]string, 0)
	for playerID := range gameCount1PlayerSeatIDs {
		gameCount1PlayerIDs = append(gameCount1PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount1PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount1PlayerIDs...)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// GameCount = 1, P1 & P2 players are playing
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount1, expectedPlayerPositions_GameCount1, sm)

	// GameCount = 2, add P3
	err = sm.AssignSeats(gameCount2PlayerSeatIDs)
	assert.NoError(t, err)

	// join P3
	gameCount2PlayerIDs := make([]string, 0)
	for playerID := range gameCount2PlayerSeatIDs {
		gameCount2PlayerIDs = append(gameCount2PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount2PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount2PlayerIDs...)

	err = sm.RotatePositions()
	assert.NoError(t, err)

	DebugPrintSeats("GameCount = 2", sm)

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount2, expectedPlayerPositions_GameCount2, sm)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}
}

func TestDefaultRule_RotatePositions_HU_NextRound_AddPlayersMixBetweenDealerBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	allPlayers := make([]string, 0)
	gameCount1PlayerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	expectedSeatPositions_GameCount1 := map[string]int{
		Position_Dealer: 3,
		Position_SB:     3,
		Position_BB:     0,
	}
	expectedPlayerPositions_GameCount1 := map[string][]string{
		"P1": {Position_BB},
		"P2": {Position_Dealer, Position_SB},
	}

	gameCount2PlayerSeatIDs := map[string]int{
		"P3": 6,
		"P4": 2,
	}
	expectedSeatPositions_GameCount2 := map[string]int{
		Position_Dealer: 6,
		Position_SB:     0,
		Position_BB:     2,
	}
	expectedPlayerPositions_GameCount2 := map[string][]string{
		"P1": {Position_SB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_BB},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(gameCount1PlayerSeatIDs)
	assert.NoError(t, err)

	// join all players
	gameCount1PlayerIDs := make([]string, 0)
	for playerID := range gameCount1PlayerSeatIDs {
		gameCount1PlayerIDs = append(gameCount1PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount1PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount1PlayerIDs...)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// GameCount = 1, P1 & P2 players are playing
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount1, expectedPlayerPositions_GameCount1, sm)

	// GameCount = 2, add P3
	err = sm.AssignSeats(gameCount2PlayerSeatIDs)
	assert.NoError(t, err)

	// join P3
	gameCount2PlayerIDs := make([]string, 0)
	for playerID := range gameCount2PlayerSeatIDs {
		gameCount2PlayerIDs = append(gameCount2PlayerIDs, playerID)
	}
	err = sm.JoinPlayers(gameCount2PlayerIDs)
	assert.NoError(t, err)
	allPlayers = append(allPlayers, gameCount2PlayerIDs...)

	err = sm.RotatePositions()
	assert.NoError(t, err)

	DebugPrintSeats("GameCount = 2", sm)

	verifySeatsAndPlayerPositions(t, expectedSeatPositions_GameCount2, expectedPlayerPositions_GameCount2, sm)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, allPlayers, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}
}

func TestDefaultRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_ValidDealerSBBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	expectedSeatPositions := []map[string]int{
		// game count = 1 (only care about dealer, sb & bb)
		{
			Position_Dealer: 4, // P3
			Position_SB:     7, // P4
			Position_BB:     0, // P1
		},
		// game count = 2 (only care about dealer, sb & bb)
		{
			Position_Dealer: 7, // P4
			Position_SB:     0, // P1
			Position_BB:     3, // P2
		},
		// game count = 3 (only care about dealer, sb & bb)
		{
			Position_Dealer: 0, // P1
			Position_SB:     3, // P2
			Position_BB:     4, // P3
		},
		// game count = 4 (only care about dealer, sb & bb)
		{
			Position_Dealer: 3, // P2
			Position_SB:     4, // P3
			Position_BB:     7, // P4
		},
		// game count = 5 (only care about dealer, sb & bb)
		{
			Position_Dealer: 4, // P3
			Position_SB:     7, // P4
			Position_BB:     0, // P1
		},
	}
	expectedPlayerPositions := []map[string][]string{
		// game count = 1 (only care about dealer, sb & bb)
		{
			"P1": {Position_BB},
			"P2": {},
			"P3": {Position_Dealer},
			"P4": {Position_SB},
		},
		// game count = 2 (only care about dealer, sb & bb)
		{
			"P1": {Position_SB},
			"P2": {Position_BB},
			"P3": {},
			"P4": {Position_Dealer},
		},
		// game count = 3 (only care about dealer, sb & bb)
		{
			"P1": {Position_Dealer},
			"P2": {Position_SB},
			"P3": {Position_BB},
			"P4": {},
		},
		// game count = 4 (only care about dealer, sb & bb)
		{
			"P1": {},
			"P2": {Position_Dealer},
			"P3": {Position_SB},
			"P4": {Position_BB},
		},
		// game count = 5 (only care about dealer, sb & bb)
		{
			"P1": {Position_BB},
			"P2": {},
			"P3": {Position_Dealer},
			"P4": {Position_SB},
		},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	for i := 0; i < 5; i++ {
		gameCount := i + 1
		expectedSeatPosition := expectedSeatPositions[i]
		expectedPlayerPosition := expectedPlayerPositions[i]

		var err error
		if gameCount == 1 {
			err = sm.InitPositions(false)
			assert.NoError(t, err)
			assert.True(t, sm.IsInitPositions())
		} else {
			err = sm.RotatePositions()
			assert.NoError(t, err)
		}

		verifySeatsAndPlayerPositions(t, expectedSeatPosition, expectedPlayerPosition, sm)
	}
}

func TestDefaultRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_ValidBBEmptyDealerSB_PlayersAreNotRemoved(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	var expectedSeatPositions map[string]int
	var expectedPlayerPositions map[string][]string

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// game count = 1 (P1, P2, P3, P4 are playing)
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	// DebugPrintSeats("game count = 1", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 4, // P3
		Position_SB:     7, // P4
		Position_BB:     0, // P1
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_BB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_SB},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)

	// game count = 2 (P1, P4 are out, P5, P6, P7 are in: P2, P3, P5, P6, P7 are playing.)
	assert.NoError(t, sm.UpdatePlayerHasChips("P1", false))
	assert.NoError(t, sm.UpdatePlayerHasChips("P4", false))

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// P5 is not between Dealer & BB but P6 & P7 are
	assert.False(t, sm.IsPlayerBetweenDealerBB("P5"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P6"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P7"))

	// join players
	err = sm.JoinPlayers([]string{"P5", "P6", "P7"})
	assert.NoError(t, err)

	// DebugPrintSeats("before game count = 2", sm)

	// P1 & P4 have no chips and P6, P7 are between Dealer & BB, so current final active players are P2, P3, P5
	assert.False(t, sm.Seats()[0].Active()) // P1
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.False(t, sm.Seats()[6].Active()) // P7
	assert.False(t, sm.Seats()[7].Active()) // P4
	assert.False(t, sm.Seats()[8].Active()) // P6

	err = sm.RotatePositions()
	assert.NoError(t, err)

	// P1 & P4 have no chips, P6 are between Dealer & BB and P5/P7 is not between Dealer & BB, so current final active players are P2, P3, P5, P7
	assert.False(t, sm.Seats()[0].Active()) // P1
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.True(t, sm.Seats()[6].Active())  // P7
	assert.False(t, sm.Seats()[7].Active()) // P4
	assert.False(t, sm.Seats()[8].Active()) // P6

	// DebugPrintSeats("game count = 2", sm)

	// P6, P7 are between Dealer & BB, so active players are P2, P3, P5
	expectedSeatPositions = map[string]int{
		Position_Dealer: 7,
		Position_SB:     0,
		Position_BB:     2,
	}
	expectedPlayerPositions = map[string][]string{
		"P2": {},
		"P3": {},
		"P5": {Position_BB},
		"P7": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_ValidBBEmptyDealerSB_PlayersAreRemoved(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	var expectedSeatPositions map[string]int
	var expectedPlayerPositions map[string][]string

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// game count = 1 (P1, P2, P3, P4 are playing)
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	// DebugPrintSeats("game count = 1", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 4, // P3
		Position_SB:     7, // P4
		Position_BB:     0, // P1
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_BB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_SB},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)

	// game count = 2 (P1, P4 are out, P5, P6, P7 are in: P2, P3, P5, P6, P7 are playing.)
	outPlayerIDs := []string{"P1", "P4"}
	err = sm.RemoveSeats(outPlayerIDs)
	assert.NoError(t, err)

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// P5 is not between Dealer & BB but P6 & P7 are
	assert.False(t, sm.IsPlayerBetweenDealerBB("P5"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P6"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P7"))

	// join players
	err = sm.JoinPlayers([]string{"P5", "P6", "P7"})
	assert.NoError(t, err)

	// DebugPrintSeats("before game count = 2", sm)

	// P1 & P4 have no chips and P6, P7 are between Dealer & BB, so current final active players are P2, P3, P5
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.False(t, sm.Seats()[6].Active()) // P7
	assert.False(t, sm.Seats()[8].Active()) // P6

	err = sm.RotatePositions()
	assert.NoError(t, err)

	// P1 & P4 have no chips, P6 are between Dealer & BB and P5/P7 is not between Dealer & BB, so current final active players are P2, P3, P5, P7
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.True(t, sm.Seats()[6].Active())  // P7
	assert.False(t, sm.Seats()[8].Active()) // P6

	// DebugPrintSeats("game count = 2", sm)

	// P6, P7 are between Dealer & BB, so active players are P2, P3, P5
	expectedSeatPositions = map[string]int{
		Position_Dealer: 7,
		Position_SB:     0,
		Position_BB:     2,
	}
	expectedPlayerPositions = map[string][]string{
		"P2": {},
		"P3": {},
		"P5": {Position_BB},
		"P7": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_ValidSBBBEmptyDealer(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	var expectedSeatPositions map[string]int
	var expectedPlayerPositions map[string][]string

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// game count = 1 (P1, P2, P3, P4 are playing)
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	// DebugPrintSeats("game count = 1", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 4, // P3
		Position_SB:     7, // P4
		Position_BB:     0, // P1
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_BB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_SB},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)

	// game count = 2 (P4 are out, P5, P6, P7 are in: P1, P2, P3, P5, P6, P7 are playing)
	assert.NoError(t, sm.UpdatePlayerHasChips("P4", false))

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// P5 is not between Dealer & BB but P6 & P7 are
	assert.False(t, sm.IsPlayerBetweenDealerBB("P5"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P6"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P7"))

	// join players
	err = sm.JoinPlayers([]string{"P5", "P6", "P7"})
	assert.NoError(t, err)

	// DebugPrintSeats("before game count = 2", sm)

	// P4 has no chips and P6, P7 are between Dealer & BB, so current final active players are P1, P2, P3, P5
	assert.True(t, sm.Seats()[0].Active())  // P1
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.False(t, sm.Seats()[6].Active()) // P7
	assert.False(t, sm.Seats()[8].Active()) // P6

	err = sm.RotatePositions()
	assert.NoError(t, err)

	// P4 has no chips, P6 is between Dealer & BB and P5/P7 is not between Dealer & BB, so current final active players are P1, P2, P3, P5, P7
	assert.True(t, sm.Seats()[0].Active())  // P1
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.True(t, sm.Seats()[6].Active())  // P7
	assert.False(t, sm.Seats()[8].Active()) // P6

	// DebugPrintSeats("game count = 2", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 7,
		Position_SB:     0,
		Position_BB:     2,
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_SB},
		"P2": {},
		"P3": {},
		"P5": {Position_BB},
		"P7": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_ValidDealerBBEmptySB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	var expectedSeatPositions map[string]int
	var expectedPlayerPositions map[string][]string

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2", "P3", "P4"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	for _, seatPlayer := range sm.Seats() {
		if seatPlayer != nil {
			assert.Contains(t, playerIDs, seatPlayer.ID)
			assert.True(t, seatPlayer.Active())
		}
	}

	// game count = 1 (P1, P2, P3, P4 are playing)
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	// DebugPrintSeats("game count = 1", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 4, // P3
		Position_SB:     7, // P4
		Position_BB:     0, // P1
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_BB},
		"P2": {},
		"P3": {Position_Dealer},
		"P4": {Position_SB},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)

	// game count = 2 (P1 are out, P5, P6, P7 are in: P2, P3, P4, P5, P6, P7 are playing)
	outPlayerIDs := []string{"P1"}
	err = sm.RemoveSeats(outPlayerIDs)
	assert.NoError(t, err)

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// P5 is not between Dealer & BB but P6 & P7 are
	assert.False(t, sm.IsPlayerBetweenDealerBB("P5"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P6"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P7"))

	// join players
	err = sm.JoinPlayers([]string{"P5", "P6", "P7"})
	assert.NoError(t, err)

	// DebugPrintSeats("before game count = 2", sm)

	// P1 has no chips and P6, P7 are between Dealer & BB, so current final active players are P2, P3, P4, P5
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.False(t, sm.Seats()[6].Active()) // P7
	assert.True(t, sm.Seats()[7].Active())  // P4
	assert.False(t, sm.Seats()[8].Active()) // P6

	err = sm.RotatePositions()
	assert.NoError(t, err)

	// DebugPrintSeats("game count = 2", sm)

	// P1 has no chips, P6 is between Dealer & BB and P5/P7 is not between Dealer & BB, so current final active players are P2, P3, P4, P5, P7
	assert.True(t, sm.Seats()[2].Active())  // P5
	assert.True(t, sm.Seats()[3].Active())  // P2
	assert.True(t, sm.Seats()[4].Active())  // P3
	assert.True(t, sm.Seats()[6].Active())  // P7
	assert.True(t, sm.Seats()[7].Active())  // P4
	assert.False(t, sm.Seats()[8].Active()) // P6

	// P6, P7 are between Dealer & BB, so active players are P2, P3, P4, P5, P7
	expectedSeatPositions = map[string]int{
		Position_Dealer: 7,
		Position_SB:     0,
		Position_BB:     2,
	}
	expectedPlayerPositions = map[string][]string{
		"P2": {},
		"P3": {},
		"P4": {Position_Dealer},
		"P5": {Position_BB},
		"P7": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

func TestDefaultRule_RotatePositions_ErrUnableToRotatePositions_BeforeInitPositions(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	err = sm.RotatePositions()
	assert.ErrorIs(t, err, ErrUnableToRotatePositions)
}

func TestDefaultRule_RotatePositions_ErrUnableToRotatePositions_InsufficientActivePlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join all players
	playerIDs := []string{"P1", "P2"}
	err = sm.JoinPlayers(playerIDs)
	assert.NoError(t, err)

	err = sm.InitPositions(false)
	assert.NoError(t, err)

	err = sm.RemoveSeats([]string{"P1"})
	assert.NoError(t, err)

	err = sm.RotatePositions()
	assert.ErrorIs(t, err, ErrUnableToRotatePositions)
}

func TestDefaultRule_BeforeInitPositions_IsPlayerBetweenDealerBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	assert.False(t, sm.IsPlayerBetweenDealerBB("P1"))
	assert.False(t, sm.IsPlayerBetweenDealerBB("P2"))
	assert.False(t, sm.IsPlayerBetweenDealerBB("P3"))
	assert.False(t, sm.IsPlayerBetweenDealerBB("P4"))
}

func TestDefaultRule_AfterInitPositions_IsPlayerBetweenDealerBB(t *testing.T) {
	maxSeat := 9
	rule := Rule_Default
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// join players
	err = sm.JoinPlayers([]string{"P1", "P2", "P3", "P4"})
	assert.NoError(t, err)

	err = sm.InitPositions(false)
	assert.NoError(t, err)

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	assert.False(t, sm.IsPlayerBetweenDealerBB("P5"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P6"))
	assert.True(t, sm.IsPlayerBetweenDealerBB("P7"))
}

func verifySeatsAndPlayerPositions(t *testing.T, expectedSeatPositions map[string]int, expectedPlayerPositions map[string][]string, sm SeatManager) {
	// check seats
	assert.Equal(t, expectedSeatPositions[Position_Dealer], sm.CurrentDealerSeatID())
	assert.Equal(t, expectedSeatPositions[Position_SB], sm.CurrentSBSeatID())
	assert.Equal(t, expectedSeatPositions[Position_BB], sm.CurrentBBSeatID())

	// check player positions
	for seatID, seatPlayer := range sm.Seats() {
		if seatPlayer != nil && seatPlayer.Active() {
			playerID := seatPlayer.ID
			expectedPositions, exist := expectedPlayerPositions[playerID]
			assert.True(t, exist)

			for _, position := range expectedPositions {
				assert.Equal(t, seatID, expectedSeatPositions[position])
			}
		}
	}
}

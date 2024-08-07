package seat_manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShortDeckRule_RotatePositions_MultipleTimes_TwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_ShortDeck
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
	}
	expectedSeatPositions_OddGameCounts := map[string]int{
		Position_Dealer: 0,
		Position_SB:     UnsetSeatID,
		Position_BB:     UnsetSeatID,
	}
	expectedPlayerPositions_OddGameCounts := map[string][]string{
		"P1": {Position_Dealer},
		"P2": {},
	}
	expectedSeatPositions_EvenGameCounts := map[string]int{
		Position_Dealer: 3,
		Position_SB:     UnsetSeatID,
		Position_BB:     UnsetSeatID,
	}
	expectedPlayerPositions_EvenGameCounts := map[string][]string{
		"P1": {},
		"P2": {Position_Dealer},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// activate all players
	playerActivateSeats := map[string]bool{
		"P1": true,
		"P2": true,
	}
	err = sm.UpdateSeatPlayerActiveStates(playerActivateSeats)
	assert.NoError(t, err)

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

func TestShortDeckRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers(t *testing.T) {
	maxSeat := 9
	rule := Rule_ShortDeck
	playerSeatIDs := map[string]int{
		"P1": 0,
		"P2": 3,
		"P3": 4,
		"P4": 7,
	}
	expectedSeatPositions := []map[string]int{
		// game count = 1
		{
			Position_Dealer: 0, // P1
			Position_SB:     UnsetSeatID,
			Position_BB:     UnsetSeatID,
		},
		// game count = 2
		{
			Position_Dealer: 3, // P2
			Position_SB:     UnsetSeatID,
			Position_BB:     UnsetSeatID,
		},
		// game count = 3
		{
			Position_Dealer: 4, // P3
			Position_SB:     UnsetSeatID,
			Position_BB:     UnsetSeatID,
		},
		// game count = 4
		{
			Position_Dealer: 7, // P4
			Position_SB:     UnsetSeatID,
			Position_BB:     UnsetSeatID,
		},
		// game count = 5
		{
			Position_Dealer: 0, // P1
			Position_SB:     UnsetSeatID,
			Position_BB:     UnsetSeatID,
		},
	}
	expectedPlayerPositions := []map[string][]string{
		// game count = 1
		{
			"P1": {Position_Dealer},
			"P2": {},
			"P3": {},
			"P4": {},
		},
		// game count = 2
		{
			"P1": {},
			"P2": {Position_Dealer},
			"P3": {},
			"P4": {},
		},
		// game count = 3
		{
			"P1": {},
			"P2": {},
			"P3": {Position_Dealer},
			"P4": {},
		},
		// game count = 4
		{
			"P1": {},
			"P2": {},
			"P3": {},
			"P4": {Position_Dealer},
		},
		// game count = 5
		{
			"P1": {Position_Dealer},
			"P2": {},
			"P3": {},
			"P4": {},
		},
	}

	sm := NewSeatManager(maxSeat, rule)
	err := sm.AssignSeats(playerSeatIDs)
	assert.NoError(t, err)

	// activate all players
	playerActivateSeats := map[string]bool{
		"P1": true,
		"P2": true,
		"P3": true,
		"P4": true,
	}
	err = sm.UpdateSeatPlayerActiveStates(playerActivateSeats)
	assert.NoError(t, err)

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

func TestShortDeckRule_RotatePositions_MultipleTimes_MoreThanTwoPlayers_PlayersInOut1(t *testing.T) {
	maxSeat := 9
	rule := Rule_ShortDeck
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

	// activate all players
	playerActivateSeats := map[string]bool{
		"P1": true,
		"P2": true,
		"P3": true,
		"P4": true,
	}
	err = sm.UpdateSeatPlayerActiveStates(playerActivateSeats)
	assert.NoError(t, err)

	// game count = 1 (P1, P2, P3, P4 are playing)
	err = sm.InitPositions(false)
	assert.NoError(t, err)
	assert.True(t, sm.IsInitPositions())

	// DebugPrintSeats("game count = 1", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 0, // P1
		Position_SB:     UnsetSeatID,
		Position_BB:     UnsetSeatID,
	}
	expectedPlayerPositions = map[string][]string{
		"P1": {Position_Dealer},
		"P2": {},
		"P3": {},
		"P4": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)

	// game count = 2 (P1, P4 are out, P5, P6, P7 are in: P2, P3, P5, P6, P7 are playing)
	outPlayerIDs := []string{"P1", "P4"}
	err = sm.CancelSeats(outPlayerIDs)
	assert.NoError(t, err)

	newPlayerSeatIDs := map[string]int{
		"P5": 2,
		"P6": 8,
		"P7": 6,
	}
	err = sm.AssignSeats(newPlayerSeatIDs)
	assert.NoError(t, err)

	// activate all players
	playerActivateSeats = map[string]bool{
		"P5": true,
		"P6": true,
		"P7": true,
	}
	err = sm.UpdateSeatPlayerActiveStates(playerActivateSeats)
	assert.NoError(t, err)

	err = sm.RotatePositions()
	assert.NoError(t, err)

	// DebugPrintSeats("game count = 2", sm)

	expectedSeatPositions = map[string]int{
		Position_Dealer: 2, // P5
		Position_SB:     UnsetSeatID,
		Position_BB:     UnsetSeatID,
	}
	expectedPlayerPositions = map[string][]string{
		"P2": {},
		"P3": {},
		"P5": {Position_Dealer},
		"P6": {},
		"P7": {},
	}
	verifySeatsAndPlayerPositions(t, expectedSeatPositions, expectedPlayerPositions, sm)
}

package seat_manager

import (
	"fmt"
	"sync"

	"github.com/thoas/go-funk"
)

type seatManager struct {
	MaxSeat      int                 `json:"max_seat"`
	SeatData     map[int]*SeatPlayer `json:"seat_data"`      // key: seat_id (from 0 to MaxSeat - 1), value: seat (nil by default)
	DealerSeatID int                 `json:"dealer_seat_id"` // UnsetSeatID by default
	SBSeatID     int                 `json:"sb_seat_id"`     // UnsetSeatID by default
	BBSeatID     int                 `json:"bb_seat_id"`     // UnsetSeatID by default
	Rule         string              `json:"rule"`           // default, short_deck
	IsInit       bool                `json:"is_init"`
	mu           sync.RWMutex        `json:"-"`
}

func (sm *seatManager) GetSeatID(playerID string) (int, error) {
	for seatID, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return seatID, nil
		}
	}

	sm.printState(1, func(tag int) {
		fmt.Printf("[DEBUG#seatManager#GetSeatID#%d] playerID: %s, seatID: %d. Error: %+v\n", tag, playerID, UnsetSeatID, ErrPlayerNotFound)
	})
	return UnsetSeatID, ErrPlayerNotFound
}

func (sm *seatManager) RandomAssignSeats(playerIDs []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	seatIDs, err := sm.randomSeatIDs(len(playerIDs))
	if err != nil {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#RandomAssignSeats#%d][randomSeatIDs] count: %d. Error: %+v\n", tag, len(playerIDs), err)
		})
		return err
	}

	for i := 0; i < len(playerIDs); i++ {
		playerID := playerIDs[i]
		seatID := seatIDs[i]
		sp := sm.newSeatPlayer(playerID)
		sm.SeatData[seatID] = &sp

		var isBetweenDealerBB bool
		if sm.IsInit {
			isBetweenDealerBB = sm.IsPlayerBetweenDealerBB(playerID)
		} else {
			isBetweenDealerBB = false
		}
		sm.SeatData[seatID].IsBetweenDealerBB = isBetweenDealerBB
	}

	return nil
}

func (sm *seatManager) AssignSeats(playerSeatIDs map[string]int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	emptySeatIDs := sm.getEmptySeatIDs()
	if len(emptySeatIDs) < len(playerSeatIDs) {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#AssignSeats#%d] len(emptySeatIDs): %d, len(playerSeatIDs): %d. Error: %+v\n", tag, len(emptySeatIDs), len(playerSeatIDs), ErrNotEnoughSeats)
		})
		return ErrNotEnoughSeats
	}

	// check duplicate players & seats
	playerIDs := make(map[string]bool)
	seats := make(map[int]bool)

	for playerID, seatID := range playerSeatIDs {
		// check players
		if _, exist := playerIDs[playerID]; exist {
			return ErrDuplicatePlayers
		}
		playerIDs[playerID] = true

		// check seats
		if _, exist := seats[seatID]; exist {
			sm.printState(2, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#AssignSeats#%d] seatID: %d. Error: %+v\n", tag, seatID, ErrDuplicateSeats)
			})
			return ErrDuplicateSeats
		}

		if seatPlayer, exist := sm.SeatData[seatID]; exist && seatPlayer != nil && seatPlayer.ID != playerID {
			sm.printState(3, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#AssignSeats#%d] seatID: %d, seatPlayer.ID: %s. playerID: %s, Error: %+v\n", tag, seatID, seatPlayer.ID, playerID, ErrSeatAlreadyIsTaken)
			})
			return ErrSeatAlreadyIsTaken
		}
		seats[seatID] = true
	}

	for _, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && funk.Contains(playerIDs, seatPlayer.ID) {
			sm.printState(4, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#AssignSeats#%d] seatPlayer.ID: %s. playerIDs: %+v. Error: %+v\n", tag, seatPlayer.ID, playerIDs, ErrDuplicatePlayers)
			})
			return ErrDuplicatePlayers
		}
	}

	// assign seats to all players
	for playerID, seatID := range playerSeatIDs {
		sp := sm.newSeatPlayer(playerID)
		sm.SeatData[seatID] = &sp

		var isBetweenDealerBB bool
		if sm.IsInit {
			isBetweenDealerBB = sm.IsPlayerBetweenDealerBB(playerID)
		} else {
			isBetweenDealerBB = false
		}
		sm.SeatData[seatID].IsBetweenDealerBB = isBetweenDealerBB
	}

	return nil
}

func (sm *seatManager) RemoveSeats(playerIDs []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	occupiedSeatIDs := sm.getOccupiedPlayerSeatIDs()
	targetSeatIDs := make([]int, 0)
	for _, playerID := range playerIDs {
		seatID, exist := occupiedSeatIDs[playerID]
		if !exist {
			sm.printState(1, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#RemoveSeats#%d] playerID: %s, occupiedSeatIDs: %+v. Error: %+v\n", tag, playerID, occupiedSeatIDs, ErrPlayerNotFound)
			})
			return ErrPlayerNotFound
		}
		targetSeatIDs = append(targetSeatIDs, seatID)
	}

	for _, seatID := range targetSeatIDs {
		sm.SeatData[seatID] = nil
	}

	return nil
}

func (sm *seatManager) JoinPlayers(playerIDs []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	targetPlayerSeatIDs := make([]int, 0)
	for _, playerID := range playerIDs {
		_, seatID, err := sm.getSeatPlayer(playerID)
		if err != nil {
			sm.printState(1, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#JoinPlayers#%d][getSeatPlayer] playerID: %s, playerIDs: %+v. Error: %+v\n", tag, playerID, playerIDs, err)
			})
			return err
		}
		targetPlayerSeatIDs = append(targetPlayerSeatIDs, seatID)
	}

	for _, seatID := range targetPlayerSeatIDs {
		sm.SeatData[seatID].IsIn = true
	}

	return nil
}

func (sm *seatManager) UpdatePlayerHasChips(playerID string, hasChips bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, seatID, err := sm.getSeatPlayer(playerID)
	if err != nil {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#UpdatePlayerHasChips#%d][getSeatPlayer] playerID: %s, hasChips: %+v. Error: %+v\n", tag, playerID, hasChips, err)
		})
		return err
	}

	sm.SeatData[seatID].HasChips = hasChips
	return nil
}

func (sm *seatManager) InitPositions(isRandom bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !funk.Contains(SupportedRules, sm.Rule) {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#InitPositions#%d] sm.Rule: %s, SupportedRules: %+v. Error: %+v\n", tag, sm.Rule, SupportedRules, ErrUnableToInitPositions)
		})
		return ErrUnableToInitPositions
	}

	if sm.IsInit {
		sm.printState(2, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#InitPositions#%d] sm.IsInit: %+v. Error: %+v\n", tag, sm.IsInit, ErrAlreadyInitPositions)
		})
		return ErrAlreadyInitPositions
	}

	if err := sm.initPositions(isRandom); err != nil {
		return err
	}

	sm.IsInit = true
	return nil
}

func (sm *seatManager) RotatePositions() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.IsInit {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#RotatePositions#%d] sm.IsInit: %+v. Error: %+v\n", tag, sm.IsInit, ErrUnableToRotatePositions)
		})
		return ErrUnableToRotatePositions
	}

	return sm.rotatePositions()
}

func (sm *seatManager) IsPlayerBetweenDealerBB(playerID string) bool {
	if !sm.IsInit {
		return false
	}

	if sm.Rule == Rule_ShortDeck {
		return false
	}

	for seatID, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return sm.isBetweenDealerBB(sm.CurrentDealerSeatID(), sm.CurrentBBSeatID(), seatID)
		}
	}

	return false
}

func (sm *seatManager) Seats() map[int]*SeatPlayer {
	return sm.SeatData
}

func (sm *seatManager) CurrentDealerSeatID() int {
	return sm.DealerSeatID
}

func (sm *seatManager) CurrentSBSeatID() int {
	return sm.SBSeatID
}

func (sm *seatManager) CurrentBBSeatID() int {
	return sm.BBSeatID
}

func (sm *seatManager) IsInitPositions() bool {
	return sm.IsInit
}

func (sm *seatManager) IsPlayerActive(playerID string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, seatID, err := sm.getSeatPlayer(playerID)
	if err != nil {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#IsPlayerActive#%d][getSeatPlayer] playerID: %s. Error: %+v\n", tag, playerID, err)
		})
		return false, err
	}

	return sm.SeatData[seatID].Active(), nil
}

func (sm *seatManager) ListPlayerSeatsFromDealer() []*SeatPlayer {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	seatPlayers := make([]*SeatPlayer, 0)
	for i := sm.DealerSeatID; i < sm.MaxSeat+sm.DealerSeatID; i++ {
		seatID := i % sm.MaxSeat
		seatPlayers = append(seatPlayers, sm.SeatData[seatID])
	}

	return seatPlayers
}

func (sm *seatManager) IsHU() bool {
	/*
		HU conditions
		1. dealer_seat is equal to sb_seat
		2. dealer_seat is not equal to sb_seat
	*/
	return sm.CurrentDealerSeatID() == sm.CurrentSBSeatID() && sm.CurrentBBSeatID() != sm.CurrentDealerSeatID()
}

package seat_manager

import (
	"sync"

	"github.com/thoas/go-funk"
)

type seatManager struct {
	maxSeat         int
	seats           map[int]*SeatPlayer // key: seat_id (from 0 to MaxSeat - 1), value: seat (nil by default)
	dealerSeatID    int                 // UnsetSeatID by default
	sbSeatID        int                 // UnsetSeatID by default
	bbSeatID        int                 // UnsetSeatID by default
	rule            string              // default, short_deck
	isInitPositions bool
	mu              sync.RWMutex
}

func (sm *seatManager) GetSeatID(playerID string) (int, error) {
	for seatID, seatPlayer := range sm.seats {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return seatID, nil
		}
	}
	return UnsetSeatID, ErrPlayerNotFound
}

func (sm *seatManager) RandomAssignSeats(playerIDs []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	seatIDs, err := sm.randomSeatIDs(len(playerIDs))
	if err != nil {
		return err
	}

	for i := 0; i < len(playerIDs); i++ {
		playerID := playerIDs[i]
		seatID := seatIDs[i]
		sp := sm.newSeatPlayer(playerID)
		sm.seats[seatID] = &sp

		var isBetweenDealerBB bool
		if sm.isInitPositions {
			isBetweenDealerBB = sm.IsPlayerBetweenDealerBB(playerID)
		} else {
			isBetweenDealerBB = false
		}
		sm.seats[seatID].IsBetweenDealerBB = isBetweenDealerBB
	}

	return nil
}

func (sm *seatManager) AssignSeats(playerSeatIDs map[string]int) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	emptySeatIDs := sm.getEmptySeatIDs()

	if len(emptySeatIDs) < len(playerSeatIDs) {
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
			return ErrDuplicateSeats
		}

		if seatPlayer, exist := sm.seats[seatID]; exist && seatPlayer != nil && seatPlayer.ID != playerID {
			return ErrSeatAlreadyIsTaken
		}
		seats[seatID] = true
	}

	for _, seatPlayer := range sm.seats {
		if seatPlayer != nil && funk.Contains(playerIDs, seatPlayer.ID) {
			return ErrDuplicatePlayers
		}
	}

	// assign seats to all players
	for playerID, seatID := range playerSeatIDs {
		sp := sm.newSeatPlayer(playerID)
		sm.seats[seatID] = &sp

		var isBetweenDealerBB bool
		if sm.isInitPositions {
			isBetweenDealerBB = sm.IsPlayerBetweenDealerBB(playerID)
		} else {
			isBetweenDealerBB = false
		}
		sm.seats[seatID].IsBetweenDealerBB = isBetweenDealerBB
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
			return ErrPlayerNotFound
		}
		targetSeatIDs = append(targetSeatIDs, seatID)
	}

	for _, seatID := range targetSeatIDs {
		sm.seats[seatID] = nil
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
			return err
		}
		targetPlayerSeatIDs = append(targetPlayerSeatIDs, seatID)
	}

	for _, seatID := range targetPlayerSeatIDs {
		sm.seats[seatID].IsIn = true
	}

	return nil
}

func (sm *seatManager) UpdatePlayerHasChips(playerID string, hasChips bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, seatID, err := sm.getSeatPlayer(playerID)
	if err != nil {
		return err
	}

	sm.seats[seatID].HasChips = hasChips
	return nil
}

func (sm *seatManager) InitPositions(isRandom bool) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !funk.Contains(SupportedRules, sm.rule) {
		return ErrUnableToInitPositions
	}

	if sm.isInitPositions {
		return ErrAlreadyInitPositions
	}

	if err := sm.initPositions(isRandom); err != nil {
		return err
	}

	sm.isInitPositions = true
	return nil
}

func (sm *seatManager) RotatePositions() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.isInitPositions {
		return ErrUnableToRotatePositions
	}

	return sm.rotatePositions()
}

func (sm *seatManager) IsPlayerBetweenDealerBB(playerID string) bool {
	if !sm.isInitPositions {
		return false
	}

	if sm.rule == Rule_ShortDeck {
		return false
	}

	for seatID, seatPlayer := range sm.seats {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return sm.isBetweenDealerBB(sm.CurrentDealerSeatID(), sm.CurrentBBSeatID(), seatID)
		}
	}

	return false
}

func (sm *seatManager) Seats() map[int]*SeatPlayer {
	return sm.seats
}

func (sm *seatManager) CurrentDealerSeatID() int {
	return sm.dealerSeatID
}

func (sm *seatManager) CurrentSBSeatID() int {
	return sm.sbSeatID
}

func (sm *seatManager) CurrentBBSeatID() int {
	return sm.bbSeatID
}

func (sm *seatManager) IsInitPositions() bool {
	return sm.isInitPositions
}

func (sm *seatManager) IsPlayerActive(playerID string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	_, seatID, err := sm.getSeatPlayer(playerID)
	if err != nil {
		return false, err
	}

	return sm.seats[seatID].Active(), nil
}

func (sm *seatManager) ListPlayerSeatsFromDealer() []*SeatPlayer {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	seatPlayers := make([]*SeatPlayer, 0)
	for i := sm.dealerSeatID; i < sm.maxSeat+sm.dealerSeatID; i++ {
		seatID := i % sm.maxSeat
		seatPlayers = append(seatPlayers, sm.seats[seatID])
	}

	return seatPlayers
}

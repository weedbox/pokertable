package seat_manager

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func (sm *seatManager) randomSeatIDs(count int) ([]int, error) {
	emptySeatIDs := sm.getEmptySeatIDs()

	if len(emptySeatIDs) < count {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#randomSeatIDs#%d] count: %d, emptySeatIDs: %+v, len(emptySeatIDs): %d. Error: %+v\n", tag, count, emptySeatIDs, len(emptySeatIDs), ErrNotEnoughSeats)
		})
		return nil, ErrNotEnoughSeats
	}

	r := sm.newRandom()
	r.Shuffle(len(emptySeatIDs), func(i, j int) {
		emptySeatIDs[i], emptySeatIDs[j] = emptySeatIDs[j], emptySeatIDs[i]
	})

	return emptySeatIDs[:count], nil
}

func (sm *seatManager) isBetweenDealerBB(dealerSeatID, bbSeatID, targetSeatID int) bool {
	if sm.Rule == Rule_ShortDeck {
		return false
	}

	if bbSeatID-dealerSeatID < 0 {
		for i := dealerSeatID + 1; i < (bbSeatID + sm.MaxSeat); i++ {
			if i%sm.MaxSeat == targetSeatID {
				return true
			}
		}
	}

	return targetSeatID < bbSeatID && targetSeatID > dealerSeatID
}

func (sm *seatManager) getEmptySeatIDs() []int {
	emptySeatIDs := make([]int, 0)
	for seatID, seatPlayer := range sm.SeatData {
		if seatPlayer == nil {
			emptySeatIDs = append(emptySeatIDs, seatID)
		}
	}
	return emptySeatIDs
}

func (sm *seatManager) getOccupiedSeatIDs() []int {
	seatIDs := make([]int, 0)
	for seatID, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && seatPlayer.Active() {
			seatIDs = append(seatIDs, seatID)
		}
	}
	sort.Slice(seatIDs, func(i, j int) bool {
		return seatIDs[i] < seatIDs[j]
	})

	return seatIDs
}

func (sm *seatManager) getOccupiedPlayerSeatIDs() map[string]int {
	seats := make(map[string]int)
	for seatID, seatPlayer := range sm.SeatData {
		if seatPlayer != nil {
			seats[seatPlayer.ID] = seatID
		}
	}
	return seats
}

func (sm *seatManager) getSeatPlayer(playerID string) (*SeatPlayer, int, error) {
	for seat, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return seatPlayer, seat, nil
		}
	}
	return nil, UnsetSeatID, ErrPlayerNotFound
}

func (sm *seatManager) randomOccupiedSeat() (int, error) {
	seatIDs := sm.getOccupiedSeatIDs()
	if len(seatIDs) == 0 {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#randomOccupiedSeat#%d] len(seatIDs): %d, seatIDs: %+v. Error: %+v\n", tag, len(seatIDs), seatIDs, ErrNotEnoughSeats)
		})
		return UnsetSeatID, ErrNotEnoughSeats
	}

	r := sm.newRandom()
	r.Shuffle(len(seatIDs), func(i, j int) {
		seatIDs[i], seatIDs[j] = seatIDs[j], seatIDs[i]
	})

	return seatIDs[0], nil
}

func (sm *seatManager) firstOccupiedSeat() (int, error) {
	seatIDs := sm.getOccupiedSeatIDs()
	if len(seatIDs) == 0 {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#firstOccupiedSeat#%d] len(seatIDs): %d, seatIDs: %+v. Error: %+v\n", tag, len(seatIDs), seatIDs, ErrNotEnoughSeats)
		})
		return UnsetSeatID, ErrNotEnoughSeats
	}
	return seatIDs[0], nil
}

func (sm *seatManager) newRandom() *rand.Rand {
	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	return rand.New(source)
}

func (sm *seatManager) nextOccupiedSeatID(startSeatID int) int {
	for i := 1; i < sm.MaxSeat; i++ {
		seatID := (startSeatID + i) % 9
		if sp, exist := sm.SeatData[seatID]; exist && sp != nil && sp.Active() {
			return seatID
		}
	}
	sm.printState(1, func(tag int) {
		fmt.Printf("[DEBUG#seatManager#nextOccupiedSeatID#%d] startSeatID: %d. UnsetSeatID Error.\n", tag, startSeatID)
	})
	return UnsetSeatID
}

func (sm *seatManager) nextInAndHasChipsSeatID(startSeatID int) int {
	for i := 1; i < sm.MaxSeat; i++ {
		seatID := (startSeatID + i) % 9
		if sp, exist := sm.SeatData[seatID]; exist && sp != nil && sp.HasChips && sp.IsIn {
			return seatID
		}
	}
	sm.printState(1, func(tag int) {
		fmt.Printf("[DEBUG#seatManager#nextInAndHasChipsSeatID#%d] startSeatID: %d. UnsetSeatID Error.\n", tag, startSeatID)
	})
	return UnsetSeatID
}

func (sm *seatManager) previousOccupiedSeatID(startSeatID int, shouldActive bool) int {
	for i := 1; i < sm.MaxSeat; i++ {
		seatID := (startSeatID + 9 - i) % 9
		if sp, exist := sm.SeatData[seatID]; exist && sp != nil {
			if shouldActive && sp.Active() {
				return seatID
			}

			if !shouldActive {
				return seatID
			}
		}
	}
	sm.printState(1, func(tag int) {
		fmt.Printf("[DEBUG#seatManager#previousOccupiedSeatID#%d] startSeatID: %d, shouldActive: %+v. UnsetSeatID Error.\n", tag, startSeatID, shouldActive)
	})
	return UnsetSeatID
}

func (sm *seatManager) previousOccupiedAliveSeatID(startSeatID int) int {
	for i := 1; i < sm.MaxSeat; i++ {
		seatID := (startSeatID + 9 - i) % 9
		if sp, exist := sm.SeatData[seatID]; exist && sp != nil {
			if sp.IsIn && sp.HasChips {
				return seatID
			}
		}
	}
	sm.printState(1, func(tag int) {
		fmt.Printf("[DEBUG#seatManager#previousOccupiedAliveSeatID#%d] startSeatID: %d. UnsetSeatID Error.\n", tag, startSeatID)
	})
	return UnsetSeatID
}

func (sm *seatManager) getActivePlayerCount() int {
	count := 0
	for _, seatPlayer := range sm.SeatData {
		if seatPlayer != nil && seatPlayer.Active() {
			count++
		}
	}
	return count
}

/*
- 2 人常牌
  - 隨機挑選一個位置當作 BB
  - Dealer & SB 為另外相同一個座位

- 超過 2 人常牌
  - 隨機挑選一個位置當作 BB
  - BB 前一個有坐人的玩家當作 SB
  - SB 前一個有坐人的玩家當作 Dealer

- 短牌
  - 隨機挑選一個位置當作 Dealer
*/
func (sm *seatManager) initPositions(isRandom bool) error {
	activeCount := sm.getActivePlayerCount()
	if activeCount < 2 {
		sm.printState(1, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#initPositions#%d] activeCount: %d. Error: %+v\n", tag, activeCount, ErrUnableToInitPositions)
		})
		return ErrUnableToInitPositions
	}

	var firstSeatID int
	if isRandom {
		seatID, err := sm.randomOccupiedSeat()
		if err != nil {
			sm.printState(2, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#initPositions#%d] [randomOccupiedSeat] seatID: %d. Error: %+v\n", tag, seatID, err)
			})
			return err
		}
		firstSeatID = seatID
	} else {
		seatID, err := sm.firstOccupiedSeat()
		if err != nil {
			sm.printState(3, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#initPositions#%d] [firstOccupiedSeat] seatID: %d. Error: %+v\n", tag, seatID, err)
			})
			return err
		}
		firstSeatID = seatID
	}

	if sm.Rule == Rule_Default {
		// pick an occupied seat id as BB
		sm.BBSeatID = firstSeatID
		if activeCount == 2 {
			// Dealer & SB are another seat id
			for seatID, seatPlayer := range sm.SeatData {
				if seatPlayer != nil && seatPlayer.Active() && seatID != firstSeatID {
					sm.DealerSeatID = seatID
					sm.SBSeatID = seatID
					break
				}
			}
		} else {
			if sbSeatID := sm.previousOccupiedSeatID(sm.BBSeatID, true); sbSeatID != UnsetSeatID {
				sm.SBSeatID = sbSeatID
			} else {
				sm.printState(4, func(tag int) {
					fmt.Printf("[DEBUG#seatManager#initPositions#%d] [previousOccupiedSeatID] sbSeatID: %d. Error: %+v\n", tag, sbSeatID, ErrUnableToInitPositions)
				})
				return ErrUnableToInitPositions
			}

			if dealerSeatID := sm.previousOccupiedSeatID(sm.SBSeatID, true); dealerSeatID != UnsetSeatID {
				sm.DealerSeatID = dealerSeatID
			} else {
				sm.printState(5, func(tag int) {
					fmt.Printf("[DEBUG#seatManager#initPositions#%d] [previousOccupiedSeatID] dealerSeatID: %d. Error: %+v\n", tag, dealerSeatID, ErrUnableToInitPositions)
				})
				return ErrUnableToInitPositions
			}
		}
	} else if sm.Rule == Rule_ShortDeck {
		sm.DealerSeatID = firstSeatID
		sm.SBSeatID = UnsetSeatID
		sm.BBSeatID = UnsetSeatID
	} else {
		// FIXME: apply more rule calculations
		sm.printState(6, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#initPositions#%d] sm.Rule: %s. Error: %+v\n", tag, sm.Rule, ErrUnableToInitPositions)
		})
		return ErrUnableToInitPositions
	}

	return nil
}

/*
- 2 人常牌
  - 新的 BB 必須要從原本 BB 往後尋找到第一個有籌碼的玩家
  - 新的 Dealer & SB 為另外相同一個座位

- 超過 2 人常牌
  - 新的 BB 必須要從原本 BB 往下家尋找到第一個有籌碼的玩家
  - 新的 SB 為上一次 BB，如果上一次 BB 玩家沒籌碼或不在位置上，新的 SB 依然是這個位置
  - 新的 Dealer 為上一次 SB，如果上一次 SB 玩家沒籌碼或不在位置上，新的 Dealer 依然是這個位置

- 短牌
  - Dealer 往下一個座位找，直到找到有籌碼的玩家為止
*/
func (sm *seatManager) rotatePositions() error {
	previousRoundIsHU := sm.IsHU()

	if sm.Rule == Rule_Default {
		previousSBSeatID := sm.SBSeatID
		previousBBSeatID := sm.BBSeatID

		// decide new bb first
		newBBSeatID := sm.nextInAndHasChipsSeatID(previousBBSeatID)
		tempNewDealerSeatID := previousSBSeatID

		// update seat_player.IsBetweenDealerBB before
		for seatID, sp := range sm.Seats() {
			if sp != nil && !sp.Active() {
				sm.SeatData[seatID].IsBetweenDealerBB = sm.isBetweenDealerBB(tempNewDealerSeatID, newBBSeatID, seatID)
			}
		}

		activeCount := sm.getActivePlayerCount()
		if activeCount < 2 {
			sm.printState(1, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#rotatePositions#%d] activeCount: %d. Error: %+v\n", tag, activeCount, ErrUnableToRotatePositions)
			})
			return ErrUnableToRotatePositions
		}

		// update bb seat id
		sm.BBSeatID = newBBSeatID
		if activeCount == 2 {
			// calc & update dealer & sb seat ids
			sm.DealerSeatID = sm.nextOccupiedSeatID(sm.BBSeatID)
			sm.SBSeatID = sm.DealerSeatID
		} else {
			// calc & update dealer & sb seat ids
			sm.SBSeatID = previousBBSeatID

			if previousRoundIsHU {
				tempNewDealerSeatID = sm.previousOccupiedAliveSeatID(sm.SBSeatID)

				// update seat_player.IsBetweenDealerBB before
				for seatID, sp := range sm.Seats() {
					if sp != nil && !sp.Active() {
						sm.SeatData[seatID].IsBetweenDealerBB = sm.isBetweenDealerBB(tempNewDealerSeatID, newBBSeatID, seatID)
					}
				}
				sm.DealerSeatID = tempNewDealerSeatID
			} else {
				sm.DealerSeatID = previousSBSeatID
			}
		}
	} else if sm.Rule == Rule_ShortDeck {
		if sm.getActivePlayerCount() < 2 {
			sm.printState(2, func(tag int) {
				fmt.Printf("[DEBUG#seatManager#rotatePositions#%d] sm.getActivePlayerCount: %d. Error: %+v\n", tag, sm.getActivePlayerCount(), ErrUnableToRotatePositions)
			})
			return ErrUnableToRotatePositions
		}

		// must find a next valid dealer
		sm.DealerSeatID = sm.nextOccupiedSeatID(sm.DealerSeatID)
		sm.SBSeatID = UnsetSeatID
		sm.BBSeatID = UnsetSeatID
	} else {
		// FIXME: apply more rule calculations
		sm.printState(3, func(tag int) {
			fmt.Printf("[DEBUG#seatManager#rotatePositions#%d] sm.Rule: %s. Error: %+v\n", tag, sm.Rule, ErrUnableToRotatePositions)
		})
		return ErrUnableToRotatePositions
	}

	return nil
}

func (sm *seatManager) newSeatPlayer(playerID string) SeatPlayer {
	return SeatPlayer{
		ID:       playerID,
		IsIn:     false,
		HasChips: true,
	}
}

func (sm *seatManager) printState(tag int, errorLogger func(tag int)) {
	errorLogger(tag)
	encoded, err := json.Marshal(sm)
	if err != nil {
		fmt.Println("[DEBUG#seatManager#printState] Error:", err)
	} else {
		fmt.Println("[DEBUG#seatManager#printState] State:", string(encoded))
	}
}

package seat_manager

import (
	"math/rand"
	"sort"
	"time"
)

func (sm *seatManager) randomSeatIDs(count int) ([]int, error) {
	emptySeatIDs := sm.getEmptySeatIDs()

	if len(emptySeatIDs) < count {
		return nil, ErrNotEnoughSeats
	}

	r := sm.newRandom()
	r.Shuffle(len(emptySeatIDs), func(i, j int) {
		emptySeatIDs[i], emptySeatIDs[j] = emptySeatIDs[j], emptySeatIDs[i]
	})

	return emptySeatIDs[:count], nil
}

func (sm *seatManager) isBetweenDealerBB(dealerSeatID, bbSeatID, targetSeatID int) bool {
	if sm.rule == Rule_ShortDeck {
		return false
	}

	if bbSeatID-dealerSeatID < 0 {
		for i := dealerSeatID + 1; i < (bbSeatID + sm.maxSeat); i++ {
			if i%sm.maxSeat == targetSeatID {
				return true
			}
		}
	}

	return targetSeatID < bbSeatID && targetSeatID > dealerSeatID
}

func (sm *seatManager) getEmptySeatIDs() []int {
	emptySeatIDs := make([]int, 0)
	for seatID, seatPlayer := range sm.seats {
		if seatPlayer == nil {
			emptySeatIDs = append(emptySeatIDs, seatID)
		}
	}
	return emptySeatIDs
}

func (sm *seatManager) getOccupiedSeatIDs() []int {
	seatIDs := make([]int, 0)
	for seatID, seatPlayer := range sm.seats {
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
	for seatID, seatPlayer := range sm.seats {
		if seatPlayer != nil {
			seats[seatPlayer.ID] = seatID
		}
	}
	return seats
}

func (sm *seatManager) getSeatPlayer(playerID string) (*SeatPlayer, int, error) {
	for seat, seatPlayer := range sm.seats {
		if seatPlayer != nil && seatPlayer.ID == playerID {
			return seatPlayer, seat, nil
		}
	}
	return nil, UnsetSeatID, ErrPlayerNotFound
}

func (sm *seatManager) randomOccupiedSeat() (int, error) {
	seatIDs := sm.getOccupiedSeatIDs()
	if len(seatIDs) == 0 {
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
	for i := 1; i < sm.maxSeat; i++ {
		seatID := (startSeatID + i) % 9
		if sp, exist := sm.seats[seatID]; exist && sp != nil && sp.Active() {
			return seatID
		}
	}
	return UnsetSeatID
}

func (sm *seatManager) nextInAndHasChipsSeatID(startSeatID int) int {
	for i := 1; i < sm.maxSeat; i++ {
		seatID := (startSeatID + i) % 9
		if sp, exist := sm.seats[seatID]; exist && sp != nil && sp.HasChips && sp.IsIn {
			return seatID
		}
	}
	return UnsetSeatID
}

func (sm *seatManager) previousOccupiedSeatID(startSeatID int, shouldActive bool) int {
	for i := 1; i < sm.maxSeat; i++ {
		seatID := (startSeatID + 9 - i) % 9
		if sp, exist := sm.seats[seatID]; exist && sp != nil {
			if shouldActive && sp.Active() {
				return seatID
			}

			if !shouldActive {
				return seatID
			}
		}
	}
	return UnsetSeatID
}

func (sm *seatManager) previousOccupiedAliveSeatID(startSeatID int) int {
	for i := 1; i < sm.maxSeat; i++ {
		seatID := (startSeatID + 9 - i) % 9
		if sp, exist := sm.seats[seatID]; exist && sp != nil {
			if sp.IsIn && sp.HasChips {
				return seatID
			}
		}
	}
	return UnsetSeatID
}

func (sm *seatManager) getActivePlayerCount() int {
	count := 0
	for _, seatPlayer := range sm.seats {
		if seatPlayer != nil && seatPlayer.Active() {
			count++
		}
	}
	return count
}

func (sm *seatManager) getPlayerCountBy(matcher func(sp *SeatPlayer) bool) int {
	count := 0
	for _, seatPlayer := range sm.seats {
		if seatPlayer != nil && matcher(seatPlayer) {
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
		return ErrUnableToInitPositions
	}

	var firstSeatID int
	if isRandom {
		seatID, err := sm.randomOccupiedSeat()
		if err != nil {
			return err
		}
		firstSeatID = seatID
	} else {
		seatID, err := sm.firstOccupiedSeat()
		if err != nil {
			return err
		}
		firstSeatID = seatID
	}

	if sm.rule == Rule_Default {
		// pick an occupied seat id as BB
		sm.bbSeatID = firstSeatID
		if activeCount == 2 {
			// Dealer & SB are another seat id
			for seatID, seatPlayer := range sm.seats {
				if seatPlayer != nil && seatPlayer.Active() && seatID != firstSeatID {
					sm.dealerSeatID = seatID
					sm.sbSeatID = seatID
					break
				}
			}
		} else {
			if sbSeatID := sm.previousOccupiedSeatID(sm.bbSeatID, true); sbSeatID != UnsetSeatID {
				sm.sbSeatID = sbSeatID
			} else {
				return ErrUnableToInitPositions
			}

			if dealerSeatID := sm.previousOccupiedSeatID(sm.sbSeatID, true); dealerSeatID != UnsetSeatID {
				sm.dealerSeatID = dealerSeatID
			} else {
				return ErrUnableToInitPositions
			}
		}
	} else if sm.rule == Rule_ShortDeck {
		sm.dealerSeatID = firstSeatID
		sm.sbSeatID = UnsetSeatID
		sm.bbSeatID = UnsetSeatID
	} else {
		// FIXME: apply more rule calculations
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

	if sm.rule == Rule_Default {
		previousSBSeatID := sm.sbSeatID
		previousBBSeatID := sm.bbSeatID

		// decide new bb first
		newBBSeatID := sm.nextInAndHasChipsSeatID(previousBBSeatID)
		tempNewDealerSeatID := previousSBSeatID

		// update seat_player.IsBetweenDealerBB before
		for seatID, sp := range sm.Seats() {
			if sp != nil && !sp.Active() {
				sm.seats[seatID].IsBetweenDealerBB = sm.isBetweenDealerBB(tempNewDealerSeatID, newBBSeatID, seatID)
			}
		}

		activeCount := sm.getActivePlayerCount()
		if activeCount < 2 {
			return ErrUnableToRotatePositions
		}

		// update bb seat id
		sm.bbSeatID = newBBSeatID
		if activeCount == 2 {
			// calc & update dealer & sb seat ids
			sm.dealerSeatID = sm.nextOccupiedSeatID(sm.bbSeatID)
			sm.sbSeatID = sm.dealerSeatID
		} else {
			// calc & update dealer & sb seat ids
			sm.sbSeatID = previousBBSeatID

			if previousRoundIsHU {
				tempNewDealerSeatID = sm.previousOccupiedAliveSeatID(sm.sbSeatID)

				// update seat_player.IsBetweenDealerBB before
				for seatID, sp := range sm.Seats() {
					if sp != nil && !sp.Active() {
						sm.seats[seatID].IsBetweenDealerBB = sm.isBetweenDealerBB(tempNewDealerSeatID, newBBSeatID, seatID)
					}
				}
				sm.dealerSeatID = tempNewDealerSeatID
			} else {
				sm.dealerSeatID = previousSBSeatID
			}
		}
	} else if sm.rule == Rule_ShortDeck {
		if sm.getActivePlayerCount() < 2 {
			return ErrUnableToRotatePositions
		}

		// must find a next valid dealer
		sm.dealerSeatID = sm.nextOccupiedSeatID(sm.dealerSeatID)
		sm.sbSeatID = UnsetSeatID
		sm.bbSeatID = UnsetSeatID
	} else {
		// FIXME: apply more rule calculations
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

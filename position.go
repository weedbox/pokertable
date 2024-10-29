package pokertable

import (
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokertable/seat_manager"
)

func NewDefaultSeatMap(seatCount int) []int {
	seatMap := make([]int, seatCount)
	for seatIdx := 0; seatIdx < seatCount; seatIdx++ {
		seatMap[seatIdx] = seat_manager.UnsetSeatID
	}
	return seatMap
}

func (te *tableEngine) updatePlayerPositions(maxSeat int, players []*TablePlayerState) {
	dealerSeatID := te.sm.CurrentDealerSeatID()
	sbSeatID := te.sm.CurrentSBSeatID()
	bbSeatID := te.sm.CurrentBBSeatID()

	playerCount := 0
	sps := te.sm.ListPlayerSeatsFromDealer()
	for i, sp := range sps {
		seatID := (i + dealerSeatID) % maxSeat

		if funk.Contains([]int{dealerSeatID, sbSeatID, bbSeatID}, seatID) {
			playerCount++
		} else {
			if sp != nil && sp.Active() {
				playerCount++
			}
		}
	}

	positions := newPositions(playerCount) // rotate & start from bb
	playerPositions := make([][]string, 0) // rotate & start from bb
	if playerCount == 2 {
		playerPositions = append(playerPositions, []string{Position_BB})
		playerPositions = append(playerPositions, []string{Position_Dealer, Position_SB})
	} else if playerCount > 2 {
		positions = rotateStringArray(positions, 2)
		for _, position := range positions {
			playerPositions = append(playerPositions, []string{position})
		}
	}

	// update player positions
	playerIdxData := make(map[string]int) // key: player_id, value: player_idx
	for playerIdx, player := range players {
		playerIdxData[player.PlayerID] = playerIdx
	}

	for i := bbSeatID; i < maxSeat+bbSeatID; i++ {
		seatID := i % maxSeat
		if seatPlayer, exist := te.sm.Seats()[seatID]; exist {
			if seatPlayer != nil && seatPlayer.Active() {
				if playerIdx, exist := playerIdxData[seatPlayer.ID]; exist && playerIdx < len(players) {
					players[playerIdx].Positions = playerPositions[0]
					playerPositions = playerPositions[1:]
				}
			} else {
				targetPosition := funk.Contains(playerPositions[0], Position_Dealer) || funk.Contains(playerPositions[0], Position_SB)
				targetSeatID := funk.Contains([]int{dealerSeatID, sbSeatID}, seatID)
				if targetPosition && targetSeatID {
					playerPositions = playerPositions[1:]
				}
			}
		}

		if len(playerPositions) == 0 {
			break
		}
	}
}

func newPositions(playerCount int) []string {
	switch playerCount {
	case 10:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_UG2,
			Position_UG3,
			Position_MP,
			Position_MP2,
			Position_HJ,
			Position_CO,
		}
	case 9:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_UG2,
			Position_MP,
			Position_MP2,
			Position_HJ,
			Position_CO,
		}
	case 8:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_UG2,
			Position_MP,
			Position_HJ,
			Position_CO,
		}
	case 7:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_MP,
			Position_HJ,
			Position_CO,
		}
	case 6:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_HJ,
			Position_CO,
		}
	case 5:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
			Position_CO,
		}
	case 4:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
			Position_UG,
		}
	case 3:
		return []string{
			Position_Dealer,
			Position_SB,
			Position_BB,
		}
	default:
		return make([]string, 0)
	}
}

/*
rotateArray 給定 source, 以 startIndex 當作第一個元素做 Rotations
  - @param source Given source array
  - @param startIndex Base index for the rotation
  - @return rotated source

Example:
  - Given: []string{"0", "1", "2", "3", "4"}, startIndex = 2
  - Output: []string{"2", "3", "4", '0', '1'}
*/
func rotateStringArray(source []string, startIndex int) []string {
	if startIndex > len(source) {
		startIndex = startIndex % len(source)
	}
	return append(source[startIndex:], source[:startIndex]...)
}

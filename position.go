package pokertable

import (
	"github.com/weedbox/pokertable/seat_manager"
)

func NewDefaultSeatMap(seatCount int) []int {
	seatMap := make([]int, seatCount)
	for seatIdx := 0; seatIdx < seatCount; seatIdx++ {
		seatMap[seatIdx] = seat_manager.UnsetSeatID
	}
	return seatMap
}

func GetPlayerPositionMap(rule string, players []*TablePlayerState, gamePlayerIndexes []int) map[int][]string {
	playerPositionMap := make(map[int][]string)
	switch rule {
	case CompetitionRule_Default, CompetitionRule_Omaha:
		positions := newPositions(len(gamePlayerIndexes))
		for gamePlayerIdx, playerIdx := range gamePlayerIndexes {
			playerPositionMap[playerIdx] = positions[gamePlayerIdx]
		}
	case CompetitionRule_ShortDeck:
		for gamePlayerIdx, playerIdx := range gamePlayerIndexes {
			if gamePlayerIdx == 0 {
				playerPositionMap[playerIdx] = []string{Position_Dealer}
			} else {
				playerPositionMap[playerIdx] = make([]string, 0)
			}
		}
	}
	return playerPositionMap
}

func newPositions(playerCount int) [][]string {
	switch playerCount {
	case 10:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_UG2},
			{Position_UG3},
			{Position_MP},
			{Position_MP2},
			{Position_HJ},
			{Position_CO},
		}
	case 9:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_UG2},
			{Position_MP},
			{Position_MP2},
			{Position_HJ},
			{Position_CO},
		}
	case 8:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_UG2},
			{Position_MP},
			{Position_HJ},
			{Position_CO},
		}
	case 7:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_MP},
			{Position_HJ},
			{Position_CO},
		}
	case 6:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_HJ},
			{Position_CO},
		}
	case 5:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
			{Position_CO},
		}
	case 4:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
			{Position_UG},
		}
	case 3:
		return [][]string{
			{Position_Dealer},
			{Position_SB},
			{Position_BB},
		}
	case 2:
		return [][]string{
			{Position_Dealer, Position_SB},
			{Position_BB},
		}
	default:
		return make([][]string, 0)
	}
}

/*
rotateIntArray 給定 source, 以 startIndex 當作第一個元素做 Rotations
  - @param source Given source array
  - @param startIndex Base index for the rotation
  - @return rotated source

Example:
  - Given: []int{0, 1, 2, 3, 4}, startIndex = 2
  - Output: []int{2, 3, 4, 0, 1}
*/
func rotateIntArray(source []int, startIndex int) []int {
	if startIndex > len(source) {
		startIndex = startIndex % len(source)
	}
	return append(source[startIndex:], source[:startIndex]...)
}

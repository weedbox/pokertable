package pokertable

import (
	"math/rand"
	"time"
)

func NewDefaultSeatMap(seatCount int) []int {
	seatMap := make([]int, seatCount)
	for seatIdx := 0; seatIdx < seatCount; seatIdx++ {
		seatMap[seatIdx] = UnsetValue
	}
	return seatMap
}

func RandomSeat(seatMap []int) int {
	emptySeats := make([]int, 0)
	for seatIdx, playerIdx := range seatMap {
		if playerIdx == UnsetValue {
			emptySeats = append(emptySeats, seatIdx)
		}
	}
	randomSeatIdx := emptySeats[randomInt(0, len(emptySeats)-1)]
	return randomSeatIdx
}

func IsBetweenDealerBB(seatIdx, currDealerTableSeatIdx, currBBTableSeatIdx, maxPlayerCount int, rule string) bool {
	if rule == CompetitionRule_ShortDeck {
		return false
	}

	if currBBTableSeatIdx-currDealerTableSeatIdx < 0 {
		for i := currDealerTableSeatIdx + 1; i < (currBBTableSeatIdx + maxPlayerCount); i++ {
			if i%maxPlayerCount == seatIdx {
				return true
			}
		}
	}

	return seatIdx < currBBTableSeatIdx && seatIdx > currDealerTableSeatIdx
}

/*
FindDealerPlayerIndex 找到 Dealer 資訊
  - @return NewDealerPlayerIndex
*/
func FindDealerPlayerIndex(gameCount, prevDealerSeatIdx, minPlayingCount, maxSeatCount int, players []*TablePlayerState, seatMap []int) int {
	newDealerIdx := UnsetValue
	if gameCount == 0 {
		// 第一次開局，隨機挑選一位玩家當 Dealer
		newDealerIdx = rand.Intn(len(players))

		// 如果玩家沒資格就照順序選有資格玩家
		if !players[newDealerIdx].IsParticipated {
			for playerIdx := 0; playerIdx < len(players); playerIdx++ {
				if players[playerIdx].IsParticipated {
					newDealerIdx = playerIdx
					break
				}
			}
		}
	} else {
		// 找下一位 Dealer Index
		for i := prevDealerSeatIdx + 1; i < (maxSeatCount + prevDealerSeatIdx + 1); i++ {
			targetTableSeatIdx := i % maxSeatCount
			targetPlayerIdx := seatMap[targetTableSeatIdx]

			if targetPlayerIdx != UnsetValue && players[targetPlayerIdx].IsParticipated && players[targetPlayerIdx].IsIn {
				newDealerIdx = targetPlayerIdx
				break
			}
		}
	}
	return newDealerIdx
}

/*
FindGamePlayerIndexes 找出參與本手的玩家 PlayerIndex 陣列
  - @return gamePlayerIndexes
  - index 0: dealer player index
  - index 1: sb player index
  - index 2: bb player index
*/
func FindGamePlayerIndexes(dealerSeatIdx int, seatMap []int, players []*TablePlayerState) []int {
	dealerPlayerIndex := seatMap[dealerSeatIdx]

	// 找出正在玩的玩家
	totalPlayersCount := 0
	gamePlayerIndexes := make([]int, 0)

	/*
		seatMapDealerPlayerIdx Dealer Player Index 在 SeatMap 中的 有參加玩家的 Index
		  - 當在 SeatMap 找有參加玩家時，如果 playerIndex == dealerPlayerIndex 則 seatMapDealerPlayerIdx 就是當前 totalPlayersCount
	*/
	seatMapDealerPlayerIdx := UnsetValue

	for _, playerIndex := range seatMap {
		if playerIndex == UnsetValue {
			continue
		}
		player := players[playerIndex]
		if player.IsParticipated {
			if playerIndex == dealerPlayerIndex {
				seatMapDealerPlayerIdx = totalPlayersCount
			}

			totalPlayersCount++
			gamePlayerIndexes = append(gamePlayerIndexes, playerIndex)
		}
	}

	if len(gamePlayerIndexes) == 0 {
		return gamePlayerIndexes
	}

	// 調整玩家陣列 Index, 以 DealerIndex 當基準當作第一個元素做 Rotations
	gamePlayerIndexes = rotateIntArray(gamePlayerIndexes, seatMapDealerPlayerIdx)

	return gamePlayerIndexes
}

func GetPlayerPositionMap(rule string, players []*TablePlayerState, gamePlayerIndexes []int) map[int][]string {
	playerPositionMap := make(map[int][]string)
	positions := newPositions(len(gamePlayerIndexes))
	for gamePlayerIdx, playerIdx := range gamePlayerIndexes {
		playerPositionMap[playerIdx] = positions[gamePlayerIdx]
	}
	// switch rule {
	// case CompetitionRule_Default, CompetitionRule_Omaha:
	// 	positions := newPositions(len(gamePlayerIndexes))
	// 	for gamePlayerIdx, playerIdx := range gamePlayerIndexes {
	// 		playerPositionMap[playerIdx] = positions[gamePlayerIdx]
	// 	}
	// case CompetitionRule_ShortDeck:
	// 	dealerPlayerIdx := gamePlayerIndexes[0]
	// 	playerPositionMap[dealerPlayerIdx] = []string{Position_Dealer}
	// }
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

func randomInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

package position

import (
	"math/rand"
	"time"

	"github.com/weedbox/pokertable/model"
	"github.com/weedbox/pokertable/util"
)

type Position struct{}

func NewPosition() Position {
	return Position{}
}

func (pos Position) NewDefaultSeatMap(seatCount int) []int {
	seatMap := make([]int, seatCount)
	for seatIdx := 0; seatIdx < seatCount; seatIdx++ {
		seatMap[seatIdx] = util.UnsetValue
	}
	return seatMap
}

func (pos Position) RandomSeatIndex(seatMap []int) int {
	emptySeatIndexes := make([]int, 0)
	for seatIdx, playerIdx := range seatMap {
		if playerIdx == util.UnsetValue {
			emptySeatIndexes = append(emptySeatIndexes, seatIdx)
		}
	}
	randomSeatIdx := emptySeatIndexes[pos.randomInt(0, len(emptySeatIndexes)-1)]
	return randomSeatIdx
}

func (pos Position) IsBetweenDealerBB(seatIdx, currDealerTableSeatIdx, currBBTableSeatIdx, maxPlayerCount int, rule string) bool {
	if rule == util.CompetitionRule_ShortDeck {
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
func (pos Position) FindDealerPlayerIndex(gameCount, prevDealerSeatIdx, minPlayingCount, maxSeatCount int, players []*model.TablePlayerState, seatMap []int) int {
	newDealerIdx := util.UnsetValue
	if gameCount == 0 {
		// 第一次開局，隨機挑選一位玩家當 Dealer
		newDealerIdx = rand.Intn(len(players))
	} else {
		// 找下一位 Dealer Index
		for i := prevDealerSeatIdx + 1; i < (maxSeatCount + prevDealerSeatIdx + 1); i++ {
			targetTableSeatIdx := i % maxSeatCount
			targetPlayerIdx := seatMap[targetTableSeatIdx]

			if targetPlayerIdx != util.UnsetValue && players[targetPlayerIdx].IsParticipated {
				newDealerIdx = targetPlayerIdx
				break
			}
		}
	}
	return newDealerIdx
}

/*
	FindPlayingPlayerIndexes 找出參與本手的玩家 PlayerIndex 陣列
	  - @return playingPlayerIndexes
	    - index 0: dealer player index
		- index 1: sb player index
		- index 2 : bb player index
*/
func (pos Position) FindPlayingPlayerIndexes(dealerSeatIdx int, seatMap []int, players []*model.TablePlayerState) []int {
	dealerPlayerIndex := seatMap[dealerSeatIdx]

	// 找出正在玩的玩家
	totalPlayersCount := 0
	playingPlayerIndexes := make([]int, 0)

	/*
		seatMapDealerPlayerIdx Dealer Player Index 在 SeatMap 中的 有參加玩家的 Index
		  - 當在 SeatMap 找有參加玩家時，如果 playerIndex == dealerPlayerIndex 則 seatMapDealerPlayerIdx 就是當前 totalPlayersCount
	*/
	seatMapDealerPlayerIdx := util.UnsetValue

	for _, playerIndex := range seatMap {
		if playerIndex == util.UnsetValue {
			continue
		}
		player := players[playerIndex]
		if player.IsParticipated {
			if playerIndex == dealerPlayerIndex {
				seatMapDealerPlayerIdx = totalPlayersCount
			}

			totalPlayersCount++
			playingPlayerIndexes = append(playingPlayerIndexes, playerIndex)
		}
	}

	// 調整玩家陣列 Index, 以 DealerIndex 當基準當作第一個元素做 Rotations
	playingPlayerIndexes = pos.rotateIntArray(playingPlayerIndexes, seatMapDealerPlayerIdx)

	return playingPlayerIndexes
}

func (pos Position) GetPlayerPositionMap(rule string, players []*model.TablePlayerState, playingPlayerIndexes []int) map[int][]string {
	playerPositionMap := make(map[int][]string)
	switch rule {
	case util.CompetitionRule_Default, util.CompetitionRule_Omaha:
		positions := pos.newPositions(len(playingPlayerIndexes))
		for idx, playerIdx := range playingPlayerIndexes {
			playerPositionMap[playerIdx] = positions[idx]
		}
	case util.CompetitionRule_ShortDeck:
		dealerPlayerIdx := playingPlayerIndexes[0]
		playerPositionMap[dealerPlayerIdx] = []string{util.Position_Dealer}
	}

	return playerPositionMap
}

func (pos Position) newPositions(playerCount int) [][]string {
	switch playerCount {
	case 9:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
			[]string{util.Position_UG1},
			[]string{util.Position_UG2},
			[]string{util.Position_UG3},
			[]string{util.Position_HJ},
			[]string{util.Position_CO},
		}
	case 8:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
			[]string{util.Position_UG1},
			[]string{util.Position_UG2},
			[]string{util.Position_HJ},
			[]string{util.Position_CO},
		}
	case 7:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
			[]string{util.Position_UG1},
			[]string{util.Position_HJ},
			[]string{util.Position_CO},
		}
	case 6:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
			[]string{util.Position_UG1},
			[]string{util.Position_CO},
		}
	case 5:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
			[]string{util.Position_CO},
		}
	case 4:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
			[]string{util.Position_UG},
		}
	case 3:
		return [][]string{
			[]string{util.Position_Dealer},
			[]string{util.Position_SB},
			[]string{util.Position_BB},
		}
	case 2:
		return [][]string{
			[]string{util.Position_Dealer, util.Position_SB},
			[]string{util.Position_BB},
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
func (pos Position) rotateIntArray(source []int, startIndex int) []int {
	if startIndex > len(source) {
		startIndex = startIndex % len(source)
	}
	return append(source[startIndex:], source[:startIndex]...)
}

func (pos Position) randomInt(min int, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

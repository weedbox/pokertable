package position

import (
	"math/rand"

	"github.com/weedbox/pokertable/pokertable/util"

	pokermodel "github.com/weedbox/pokermodel"
)

type Position struct{}

func NewPosition() Position {
	return Position{}
}

func (pos Position) NewDefaultSeatMap(seatCount int) []int {
	seatMap := make([]int, seatCount)
	for seatIdx := 0; seatIdx < seatCount; seatIdx++ {
		seatMap[seatIdx] = util.UnsetIndex
	}
	return seatMap
}

func (pos Position) RandomSeatIndex(seatMap []int) int {
	emptySeatIndexes := make([]int, 0)
	for seatIdx, playerIdx := range seatMap {
		if playerIdx == util.UnsetIndex {
			emptySeatIndexes = append(emptySeatIndexes, seatIdx)
		}
	}
	randomSeatIdx := emptySeatIndexes[util.RandomInt(0, len(emptySeatIndexes)-1)]
	return randomSeatIdx
}

func (pos Position) IsBetweenDealerBB(seatIdx, currDealerTableSeatIdx, currBBTableSeatIdx, maxPlayerCount int, rule pokermodel.CompetitionRule) bool {
	if rule == pokermodel.CompetitionRule_ShortDeck {
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
func (pos Position) FindDealerPlayerIndex(gameCount, prevDealerSeatIdx, minPlayingCount, maxSeatCount int, players []pokermodel.TablePlayerState, seatMap []int) int {
	newDealerIdx := util.UnsetIndex
	if gameCount == 0 {
		// 第一次開局，隨機挑選一位玩家當 Dealer
		newDealerIdx = rand.Intn(len(players))
	} else {
		// 找下一位 Dealer Index
		for i := prevDealerSeatIdx + 1; i < (maxSeatCount + prevDealerSeatIdx + 1); i++ {
			targetTableSeatIdx := i % maxSeatCount
			targetPlayerIdx := seatMap[targetTableSeatIdx]

			if targetPlayerIdx != util.UnsetIndex && players[targetPlayerIdx].IsParticipated {
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
func (pos Position) FindPlayingPlayerIndexes(dealerSeatIdx int, seatMap []int, players []pokermodel.TablePlayerState) []int {
	dealerPlayerIndex := seatMap[dealerSeatIdx]

	// 找出正在玩的玩家
	totalPlayersCount := 0
	playingPlayerIndexes := make([]int, 0)

	/*
		seatMapDealerPlayerIdx Dealer Player Index 在 SeatMap 中的 有參加玩家的 Index
		  - 當在 SeatMap 找有參加玩家時，如果 playerIndex == dealerPlayerIndex 則 seatMapDealerPlayerIdx 就是當前 totalPlayersCount
	*/
	seatMapDealerPlayerIdx := util.UnsetIndex

	for _, playerIndex := range seatMap {
		if playerIndex == util.UnsetIndex {
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
	playingPlayerIndexes = util.RotateIntArray(playingPlayerIndexes, seatMapDealerPlayerIdx)

	return playingPlayerIndexes
}

func (pos Position) GetPlayerPositionMap(rule string, players []pokermodel.TablePlayerState, playingPlayerIndexes []int) map[int][]string {
	playerPositionMap := make(map[int][]string)
	switch rule {
	case pokermodel.CompetitionRule_Default, pokermodel.CompetitionRule_Omaha:
		positions := pos.newPositions(len(playingPlayerIndexes))
		for idx, playerIdx := range playingPlayerIndexes {
			playerPositionMap[playerIdx] = positions[idx]
		}
	case pokermodel.CompetitionRule_ShortDeck:
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

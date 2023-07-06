package testcases

import (
	"fmt"
	"testing"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokertable"
)

func LogJSON(t *testing.T, msg string, jsonPrinter func() (string, error)) {
	json, _ := jsonPrinter()
	fmt.Printf("\n===== [%s] =====\n%s\n", msg, json)
}

func NewDefaultTableSetting(joinPlayers ...pokertable.JoinPlayer) pokertable.TableSetting {
	return pokertable.TableSetting{
		ShortID:        "ABC123",
		Code:           "01",
		Name:           "CT 20 min 01",
		InvitationCode: "come_to_play",
		CompetitionMeta: pokertable.CompetitionMeta{
			ID:                  "1005c477-84b4-4d1b-9fca-3a6ad84e0fe7",
			MaxDuration:         3,
			Rule:                pokertable.CompetitionRule_Default,
			Mode:                pokertable.CompetitionMode_CT,
			TableMaxSeatCount:   9,
			TableMinPlayerCount: 2,
			MinChipUnit:         10,
			ActionTime:          10,
		},
		JoinPlayers: joinPlayers,
	}
}

// func NewDefaultTableSetting(joinPlayers ...pokertable.JoinPlayer) pokertable.TableSetting {
// 	return pokertable.TableSetting{
// 		ShortID:        "ABC123",
// 		Code:           "01",
// 		Name:           "CT 20 min 01",
// 		InvitationCode: "come_to_play",
// 		CompetitionMeta: pokertable.CompetitionMeta{
// 			ID: "1005c477-84b4-4d1b-9fca-3a6ad84e0fe7",
// 			Blind: pokertable.Blind{
// 				ID:              uuid.New().String(),
// 				Name:            "20 min FAST",
// 				InitialLevel:    1,
// 				FinalBuyInLevel: 2,
// 				DealerBlindTime: 1,
// 				Levels: []pokertable.BlindLevel{
// 					{
// 						Level:    1,
// 						SB:       10,
// 						BB:       20,
// 						Ante:     0,
// 						Duration: 1,
// 					},
// 					{
// 						Level:    2,
// 						SB:       20,
// 						BB:       30,
// 						Ante:     0,
// 						Duration: 1,
// 					},
// 					{
// 						Level:    3,
// 						SB:       30,
// 						BB:       40,
// 						Ante:     0,
// 						Duration: 1,
// 					},
// 				},
// 			},
// 			MaxDuration:         3,
// 			Rule:                pokertable.CompetitionRule_Default,
// 			Mode:                pokertable.CompetitionMode_CT,
// 			TableMaxSeatCount:   9,
// 			TableMinPlayerCount: 2,
// 			MinChipUnit:         10,
// 			ActionTime:          10,
// 		},
// 		JoinPlayers: joinPlayers,
// 	}
// }

func currentPlayerMove(table *pokertable.Table) (string, []string) {
	playerID := ""
	currGamePlayerIdx := table.State.GameState.Status.CurrentPlayer
	for gamePlayerIdx, playerIdx := range table.State.GamePlayerIndexes {
		if gamePlayerIdx == currGamePlayerIdx {
			playerID = table.State.PlayerStates[playerIdx].PlayerID
			break
		}
	}
	return playerID, table.State.GameState.Players[currGamePlayerIdx].AllowedActions
}

func findPlayerID(table *pokertable.Table, position string) string {
	for _, playerIdx := range table.State.GamePlayerIndexes {
		player := table.State.PlayerStates[playerIdx]
		if funk.Contains(player.Positions, position) {
			return player.PlayerID
		}
	}
	return ""
}

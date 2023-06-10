package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func logJSON(t *testing.T, msg string, jsonPrinter func() (*string, error)) {
	json, _ := jsonPrinter()
	t.Logf("\n===== [%s] =====\n%s\n", msg, *json)
}

func FindCurrentPlayerID(table *pokertable.Table) string {
	currGamePlayerIdx := table.State.GameState.Status.CurrentPlayer
	for gamePlayerIdx, playerIdx := range table.State.GamePlayerIndexes {
		if gamePlayerIdx == currGamePlayerIdx {
			return table.State.PlayerStates[playerIdx].PlayerID
		}
	}
	return ""
}

func PrintPlayerActionLog(table *pokertable.Table, playerID, actionLog string) {
	findPlayerIdx := func(players []*pokertable.TablePlayerState, targetPlayerID string) int {
		for idx, player := range players {
			if player.PlayerID == targetPlayerID {
				return idx
			}
		}

		return -1
	}

	positions := make([]string, 0)
	playerIdx := findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != -1 {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	fmt.Printf("[%s] %s%+v: %s\n", table.State.GameState.Status.Round, playerID, positions, actionLog)
}

func NewPlayerActionErrorLog(table *pokertable.Table, playerID, actionLog string, err error) string {
	if err == nil {
		return ""
	}

	findPlayerIdx := func(players []*pokertable.TablePlayerState, targetPlayerID string) int {
		for idx, player := range players {
			if player.PlayerID == targetPlayerID {
				return idx
			}
		}

		return -1
	}

	positions := make([]string, 0)
	playerIdx := findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != -1 {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	return fmt.Sprintf("[%s] %s%+v: %s. Error: %s\n", table.State.GameState.Status.Round, playerID, positions, actionLog, err.Error())
}

func AllGamePlayersReady(t *testing.T, tableEngine pokertable.TableEngine, table *pokertable.Table) {
	for _, playerIdx := range table.State.GamePlayerIndexes {
		player := table.State.PlayerStates[playerIdx]
		err := tableEngine.PlayerReady(table.ID, player.PlayerID)
		assert.Nil(t, err, fmt.Sprintf("[PlayerReady] [%s] %s ready error: %+v\n", table.State.GameState.Status.Round, player.PlayerID, err))
		fmt.Printf("[PlayerReady] [%s] %s is ready. CurrentEvent: %s\n", table.State.GameState.Status.Round, player.PlayerID, table.State.GameState.Status.CurrentEvent.Name)
	}
}

func AllPlayersPlaying(t *testing.T, tableEngine pokertable.TableEngine, tableID string) {
	// game started
	// all players ready
	table, _ := tableEngine.GetTable(tableID)
	AllGamePlayersReady(t, tableEngine, table)
	// logJSON(t, fmt.Sprintf("Game %d - all players ready", table.State.GameCount), table.GetJSON)

	// preflop
	// pay sb
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pay sb")
	err := tableEngine.PlayerPaySB(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "pay sb", err))
	fmt.Printf("[PlayerPaySB] dealer receive bb.\n")

	// pay bb
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pay bb")
	err = tableEngine.PlayerPayBB(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "pay bb", err))
	fmt.Printf("[PlayerPayBB] dealer receive bb.\n")

	// rest players ready
	AllGamePlayersReady(t, tableEngine, table)
	// logJSON(t, fmt.Sprintf("Game %d - preflop all players ready", table.State.GameCount), table.GetJSON)

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "check")
	err = tableEngine.PlayerCheck(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "check", err))

	// logJSON(t, fmt.Sprintf("Game %d - preflop all players done actions", table.State.GameCount), table.GetJSON)

	// flop
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// turn
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// river
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))
}

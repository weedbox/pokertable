package testcases

import (
	"testing"

	"github.com/weedbox/pokerface"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func logJSON(t *testing.T, msg string, jsonPrinter func() (*string, error)) {
	json, _ := jsonPrinter()
	t.Logf("\n===== [%s] =====\n%s\n", msg, *json)
}

// func FindCurrentPlayerID(table pokertable.Table, currPlayerIndex int) string {
// 	for playingPlayerIndex, playerIndex := range table.State.PlayingPlayerIndexes {
// 		if playingPlayerIndex == currPlayerIndex {
// 			return table.State.PlayerStates[playerIndex].PlayerID
// 		}
// 	}
// 	return ""
// }

func FindCurrentPlayerID(table pokertable.Table) string {
	currPlayerIndex := table.State.GameState.Status.CurrentPlayer
	for playingPlayerIndex, playerIndex := range table.State.PlayingPlayerIndexes {
		if playingPlayerIndex == currPlayerIndex {
			return table.State.PlayerStates[playerIndex].PlayerID
		}
	}
	return ""
}

func AllGamePlayersReady(t *testing.T, tableEngine pokertable.TableEngine, table pokertable.Table) pokertable.Table {
	ret := table
	for _, playingPlayerIdx := range table.State.PlayingPlayerIndexes {
		player := table.State.PlayerStates[playingPlayerIdx]
		table, err := tableEngine.PlayerReady(table, player.PlayerID)
		assert.Nil(t, err)
		ret = table
	}
	return ret
}

func NextRound(t *testing.T, tableEngine pokertable.TableEngine, table pokertable.Table) pokertable.Table {
	if table.State.GameState.Status.CurrentEvent.Name == pokerface.GameEventSymbols[pokerface.GameEvent_RoundClosed] {
		newTable, err := tableEngine.NextRound(table)
		assert.Nil(t, err)
		return newTable
	}
	return table
}

func TableSettlement(t *testing.T, tableEngine pokertable.TableEngine, table pokertable.Table) pokertable.Table {
	if table.State.GameState.Status.CurrentEvent.Name == pokerface.GameEventSymbols[pokerface.GameEvent_GameClosed] {
		newTable := tableEngine.TableSettlement(table)
		return newTable
	}
	return table
}

func AllPlayersPlaying(t *testing.T, tableEngine pokertable.TableEngine, table pokertable.Table) pokertable.Table {
	// game started
	// all players ready
	newTable := AllGamePlayersReady(t, tableEngine, table)
	table = newTable
	// logJSON(t, fmt.Sprintf("Game %d - all players ready", table.State.GameCount), table.GetJSON)

	// preflop
	// pay sb
	newTable, err := tableEngine.PlayerPaySB(table, FindCurrentPlayerID(table))
	assert.Nil(t, err)
	table = newTable

	// pay bb
	newTable, err = tableEngine.PlayerPayBB(table, FindCurrentPlayerID(table))
	assert.Nil(t, err)
	table = newTable

	// rest players ready
	newTable = AllGamePlayersReady(t, tableEngine, table)
	table = newTable

	// dealer move
	newTable, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))
	assert.Nil(t, err)
	table = newTable

	// sb move
	newTable, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))
	assert.Nil(t, err)
	table = newTable

	// bb move
	newTable, err = tableEngine.PlayerCheck(table, FindCurrentPlayerID(table))
	assert.Nil(t, err)
	table = newTable

	// next round
	newTable = NextRound(t, tableEngine, table)
	table = newTable

	// flop
	// all players ready
	table = AllGamePlayersReady(t, tableEngine, table)

	// sb move
	table, err = tableEngine.PlayerBet(table, FindCurrentPlayerID(table), 10)

	// bb move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// dealer move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// next round
	newTable = NextRound(t, tableEngine, table)
	table = newTable

	// turn
	// all players ready
	table = AllGamePlayersReady(t, tableEngine, table)

	// sb move
	table, err = tableEngine.PlayerBet(table, FindCurrentPlayerID(table), 10)

	// bb move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// dealer move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// next round
	newTable = NextRound(t, tableEngine, table)
	table = newTable

	// river
	// all players ready
	table = AllGamePlayersReady(t, tableEngine, table)

	// sb move
	table, err = tableEngine.PlayerBet(table, FindCurrentPlayerID(table), 10)

	// bb move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// dealer move
	table, err = tableEngine.PlayerCall(table, FindCurrentPlayerID(table))

	// next round
	newTable = NextRound(t, tableEngine, table)
	table = newTable

	// settlement
	newTable = TableSettlement(t, tableEngine, table)
	table = newTable

	return table
}

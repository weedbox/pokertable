package testcases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func logJSON(t *testing.T, msg string, jsonPrinter func() (*string, error)) {
	json, _ := jsonPrinter()
	t.Logf("\n===== [%s] =====\n%s\n", msg, *json)
}

func FindCurrentPlayerID(table *pokertable.Table) string {
	currPlayerIndex := table.State.GameState.Status.CurrentPlayer
	for playingPlayerIndex, playerIndex := range table.State.PlayingPlayerIndexes {
		if playingPlayerIndex == currPlayerIndex {
			return table.State.PlayerStates[playerIndex].PlayerID
		}
	}
	return ""
}

func AllGamePlayersReady(t *testing.T, tableEngine pokertable.TableEngine, table *pokertable.Table) {
	for _, playingPlayerIdx := range table.State.PlayingPlayerIndexes {
		player := table.State.PlayerStates[playingPlayerIdx]
		err := tableEngine.PlayerReady(table.ID, player.PlayerID)
		assert.Nil(t, err)
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
	err := tableEngine.PlayerPaySB(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// pay bb
	err = tableEngine.PlayerPayBB(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// rest players ready
	AllGamePlayersReady(t, tableEngine, table)
	// logJSON(t, fmt.Sprintf("Game %d - preflop all players ready", table.State.GameCount), table.GetJSON)

	// dealer move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// sb move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// bb move
	err = tableEngine.PlayerCheck(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// logJSON(t, fmt.Sprintf("Game %d - preflop all players done actions", table.State.GameCount), table.GetJSON)

	// flop
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err)

	// bb move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// dealer move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// turn
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err)

	// bb move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// dealer move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// river
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	assert.Nil(t, err)

	// bb move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)

	// dealer move
	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	assert.Nil(t, err)
}

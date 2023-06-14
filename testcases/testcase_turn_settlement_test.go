package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestTableGame_Turn_Settlement(t *testing.T) {
	// player actions
	playersAutoPlay := func(tableEngine pokertable.TableEngine, tableID string) {
		// game started
		// all players ready
		table, err := tableEngine.GetTable(tableID)
		assert.Nil(t, err, "get table failed")
		AllGamePlayersReady(t, tableEngine, table)

		// preflop
		// pay sb
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pay sb")
		err = tableEngine.PlayerPaySB(tableID, FindCurrentPlayerID(table))
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

		// flop
		// all players ready
		AllGamePlayersReady(t, tableEngine, table)

		// sb move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
		err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

		// bb move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "fold")
		err = tableEngine.PlayerFold(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "fold", err))

		// dealer move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
		err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

		// turn
		// all players ready
		AllGamePlayersReady(t, tableEngine, table)

		// sb move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
		err = tableEngine.PlayerAllin(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))

		// bb move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pass")
		err = tableEngine.PlayerPass(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "pass", err))

		// dealer move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
		err = tableEngine.PlayerAllin(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))
	}

	playerIDs := []string{"Fred", "Jeffrey", "Chuck"}
	tableEngine, tableID := CreateTableAndStartGame(t, playerIDs)
	playersAutoPlay(tableEngine, tableID)
}

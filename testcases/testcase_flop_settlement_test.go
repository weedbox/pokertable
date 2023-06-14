package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestTableGame_Flop_Settlement(t *testing.T) {
	// player actions
	playersAutoPlay := func(tableEngine pokertable.TableEngine, tableID string) {
		// game started
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
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
		err = tableEngine.PlayerAllin(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))

		// bb move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
		err = tableEngine.PlayerAllin(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))

		// dealer move
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
		err = tableEngine.PlayerAllin(tableID, FindCurrentPlayerID(table))
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))
	}

	// create & start game
	playerIDs := []string{"Fred", "Jeffrey", "Chuck"}
	tableEngine, tableID := CreateTableAndStartGame(t, playerIDs)
	playersAutoPlay(tableEngine, tableID)
}

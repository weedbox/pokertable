package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestTableGame_Turn_Settlement(t *testing.T) {
	// given conditions
	playerIDs := []string{"Fred", "Jeffrey", "Chuck"}
	playersAutoPlayActions := func(tableEngine pokertable.TableEngine, tableID string) {
		// game started
		// all players ready
		table, err := tableEngine.GetTable(tableID)
		assert.Nil(t, err, "get table failed")
		AllGamePlayersReady(t, tableEngine, table)

		// preflop
		// pay sb
		sb := table.State.BlindState.CurrentBlindLevel().BlindLevel.SB
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay sb %d", sb))
		err = tableEngine.PlayerPay(tableID, FindCurrentPlayerID(table), sb)
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay sb %d", sb), err))
		fmt.Printf("[PlayerPaySB] dealer receive sb %d.\n", sb)

		// pay bb
		bb := table.State.BlindState.CurrentBlindLevel().BlindLevel.BB
		PrintPlayerActionLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay bb %d", bb))
		err = tableEngine.PlayerPay(tableID, FindCurrentPlayerID(table), bb)
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay bb %d", sb), err))
		fmt.Printf("[PlayerPaySB] dealer receive bb %d.\n", bb)

		// all players ready
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

	// create a table
	tableEngine := pokertable.NewTableEngine()
	tableEngine.OnTableUpdated(func(table *pokertable.Table) {
		switch table.State.Status {
		case pokertable.TableStateStatus_TableGameOpened:
			DebugPrintTableGameOpened(*table)
		case pokertable.TableStateStatus_TableGameSettled:
			DebugPrintTableGameSettled(*table)
		}
	})
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	// buy in
	redeemChips := int64(15000)
	players := make([]pokertable.JoinPlayer, 0)
	for _, playerID := range playerIDs {
		players = append(players, pokertable.JoinPlayer{
			PlayerID:    playerID,
			RedeemChips: redeemChips,
		})
	}
	for _, joinPlayer := range players {
		err = tableEngine.PlayerJoin(table.ID, joinPlayer)
		assert.Nil(t, err)
	}

	// start game
	err = tableEngine.StartTableGame(table.ID)
	assert.Nil(t, err)

	playersAutoPlayActions(tableEngine, table.ID)
}

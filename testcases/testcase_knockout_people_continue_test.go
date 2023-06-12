package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestTableGame_Knockout_People_Continue(t *testing.T) {
	// create a table
	tableEngine := pokertable.NewTableEngine()
	tableEngine.OnTableUpdated(func(model *pokertable.Table) {})
	tableEngine.OnTableSettled(func(model *pokertable.Table) {})
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	// buy in 2 players
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 150},
		{PlayerID: "Fred", RedeemChips: 150},
		{PlayerID: "Chuck", RedeemChips: 150},
	}
	for _, joinPlayer := range players {
		err = tableEngine.PlayerJoin(table.ID, joinPlayer)
		assert.Nil(t, err)
	}

	// start game (count = 1)
	err = tableEngine.StartGame(table.ID)
	assert.Nil(t, err)
	// logJSON(t, fmt.Sprintf("Game %d - game started", table.State.GameCount), table.GetJSON)

	// game started
	// all players ready
	table, _ = tableEngine.GetTable(table.ID)
	AllGamePlayersReady(t, tableEngine, table)
	// logJSON(t, fmt.Sprintf("Game %d - all players ready", table.State.GameCount), table.GetJSON)

	// preflop
	// pay sb
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pay sb")
	err = tableEngine.PlayerPaySB(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "pay sb", err))
	fmt.Printf("[PlayerPaySB] dealer receive bb.\n")

	// pay bb
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "pay bb")
	err = tableEngine.PlayerPayBB(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "pay bb", err))
	fmt.Printf("[PlayerPayBB] dealer receive bb.\n")

	// rest players ready
	AllGamePlayersReady(t, tableEngine, table)
	// logJSON(t, fmt.Sprintf("Game %d - preflop all players ready", table.State.GameCount), table.GetJSON)

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	err = tableEngine.PlayerCall(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "check")
	err = tableEngine.PlayerCheck(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "check", err))

	// logJSON(t, fmt.Sprintf("Game %d - preflop all players done actions", table.State.GameCount), table.GetJSON)

	// flop
	// all players ready
	AllGamePlayersReady(t, tableEngine, table)

	// sb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
	err = tableEngine.PlayerAllin(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))

	// bb move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "fold")
	err = tableEngine.PlayerFold(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "fold", err))

	// dealer move
	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "allin")
	err = tableEngine.PlayerAllin(table.ID, FindCurrentPlayerID(table))
	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "all in", err))

	// logJSON(t, fmt.Sprintf("Game %d - settlement", table.State.GameCount), table.GetJSON)
}

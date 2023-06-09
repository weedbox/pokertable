package testcases

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestBasicTableGame(t *testing.T) {
	// create a table
	gameEngine := pokertable.NewGameEngine()
	tableEngine := pokertable.NewTableEngine(gameEngine, uint32(logrus.DebugLevel))
	tableEngine.OnTableUpdated(func(model *pokertable.Table) {
		logJSON(t, "OnTableUpdated", model.GetJSON)
	})
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	// buy in 3 players
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 150},
		{PlayerID: "Chuck", RedeemChips: 150},
		{PlayerID: "Fred", RedeemChips: 150},
	}
	for _, joinPlayer := range players {
		table, err = tableEngine.PlayerJoin(table, joinPlayer)
		assert.Nil(t, err)
	}

	// start game (count = 1)
	table, err = tableEngine.StartGame(table)
	assert.Nil(t, err)

	// logJSON(t, fmt.Sprintf("game %d started:", table.State.GameCount), table.GetJSON)

	// game count 1: players playing
	table = AllPlayersPlaying(t, tableEngine, table)

	// start game (count = 2)
	table, err = tableEngine.GameOpen(table)
	assert.Nil(t, err)

	// game count 2: players playing
	_ = AllPlayersPlaying(t, tableEngine, table)
}

package testcases

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
	"github.com/weedbox/pokertable/model"
)

func TestBasicTableGame(t *testing.T) {
	// create a table
	gameEngine := pokertable.NewGameEngine()
	tableEngine := pokertable.NewTableEngine(gameEngine)
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	// buy in 3 players
	players := []model.JoinPlayer{
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

	// game count 1: players playing
	table = AllPlayersPlaying(t, tableEngine, table)

	// start game (count = 2)
	table, err = tableEngine.GameOpen(table)
	assert.Nil(t, err)

	// game count 2: players playing
	_ = AllPlayersPlaying(t, tableEngine, table)
}

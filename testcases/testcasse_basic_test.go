package testcases

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func TestBasicTableGame(t *testing.T) {
	// create a table
	tableEngine := pokertable.NewTableEngine(uint32(logrus.DebugLevel))
	tableEngine.OnTableUpdated(func(model *pokertable.Table) {})
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
		err = tableEngine.PlayerJoin(table.ID, joinPlayer)
		assert.Nil(t, err)
	}

	// start game (count = 1)
	err = tableEngine.StartGame(table.ID)
	assert.Nil(t, err)

	// logJSON(t, fmt.Sprintf("game %d started:", table.State.GameCount), table.GetJSON)

	// game count 1: players playing
	AllPlayersPlaying(t, tableEngine, table.ID)

	// game count 2: players playing
	AllPlayersPlaying(t, tableEngine, table.ID)
}

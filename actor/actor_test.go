package actor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	pokertable "github.com/weedbox/pokertable"
)

func TestActor_Basic(t *testing.T) {

	// Initializing table
	// create manager & table
	manager := pokertable.NewManager()
	table, err := manager.CreateTable(pokertable.TableSetting{
		TableID: uuid.New().String(),
		Meta: pokertable.TableMeta{
			CompetitionID:       uuid.New().String(),
			Rule:                pokertable.CompetitionRule_Default,
			Mode:                pokertable.CompetitionMode_CT,
			MaxDuration:         10,
			TableMaxSeatCount:   9,
			TableMinPlayerCount: 2,
			MinChipUnit:         10,
			ActionTime:          10,
		},
	})
	assert.Nil(t, err, "create table failed")

	// get table engine
	tableEngine, err := manager.GetTableEngine(table.ID)
	assert.Nil(t, err, "get table engine failed")

	// Initializing bot
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 3000},
		{PlayerID: "Chuck", RedeemChips: 3000},
		{PlayerID: "Fred", RedeemChips: 3000},
	}

	// Preparing actors
	actors := make([]Actor, 0)
	for _, p := range players {

		// Create new actor
		a := NewActor()

		// Initializing table engine adapter to communicate with table engine
		tc := NewTableEngineAdapter(tableEngine, table)
		a.SetAdapter(tc)

		// Initializing bot runner
		bot := NewBotRunner(p.PlayerID)
		a.SetRunner(bot)

		actors = append(actors, a)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	// Preparing table state updater
	tableEngine.OnTableErrorUpdated(func(table *pokertable.Table, err error) {
		t.Log("ERROR:", err)
	})
	tableEngine.OnTableUpdated(func(table *pokertable.Table) {
		// t.Log("UPDATED")
		// Update table state via adapter
		for _, a := range actors {
			a.GetTable().UpdateTableState(table)
		}

		switch table.State.Status {
		case pokertable.TableStateStatus_TableGameOpened:
			DebugPrintTableGameOpened(*table)
		case pokertable.TableStateStatus_TableGameSettled:
			DebugPrintTableGameSettled(*table)
		case pokertable.TableStateStatus_TablePausing:
			err := tableEngine.CloseTable()
			assert.Nil(t, err, "close table failed")
		case pokertable.TableStateStatus_TableClosed:
			t.Log("table is closed")
			wg.Done()
			return
		}
	})

	// Add player to table
	for _, p := range players {
		assert.Nil(t, tableEngine.PlayerReserve(p), fmt.Sprintf("%s reserve error", p.PlayerID))
		assert.Nil(t, tableEngine.PlayerJoin(p.PlayerID), fmt.Sprintf("%s join error", p.PlayerID))
	}

	// Start game
	tableEngine.UpdateBlind(1, 0, 0, 10, 20)
	err = tableEngine.StartTableGame()
	assert.Nil(t, err)

	wg.Wait()
}

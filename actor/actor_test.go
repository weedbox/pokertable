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
	var wg sync.WaitGroup
	var tableEngine pokertable.TableEngine

	// Initializing table
	// create manager & table
	actors := make([]Actor, 0)
	manager := pokertable.NewManager()
	tableSetting := pokertable.TableSetting{
		TableID: uuid.New().String(),
		Meta: pokertable.TableMeta{
			CompetitionID:       uuid.New().String(),
			Rule:                pokertable.CompetitionRule_Default,
			Mode:                pokertable.CompetitionMode_CT,
			MaxDuration:         10,
			TableMaxSeatCount:   9,
			TableMinPlayerCount: 2,
			MinChipUnit:         3,
			ActionTime:          12,
		},
	}
	tableEngineOption := pokertable.NewTableEngineOptions()
	tableEngineOption.Interval = 1
	tableEngineCallbacks := pokertable.NewTableEngineCallbacks()
	tableEngineCallbacks.OnTableUpdated = func(table *pokertable.Table) {
		if table.State.LastPlayerGameAction != nil {
			fmt.Printf("[#%d][%s][%s][%s, %d][%+v], Seat: %d\n",
				table.State.LastPlayerGameAction.GameCount,
				table.State.LastPlayerGameAction.Round,
				table.State.LastPlayerGameAction.PlayerID,
				table.State.LastPlayerGameAction.Action,
				table.State.LastPlayerGameAction.Chips,
				table.State.LastPlayerGameAction.Positions,
				table.State.LastPlayerGameAction.Seat,
			)
		}

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
	}
	tableEngineCallbacks.OnTableErrorUpdated = func(table *pokertable.Table, err error) {
		t.Log("[Table] Error:", err)
	}
	tableEngineCallbacks.OnAutoGameOpenEnd = func(competitionID, tableID string) {
		t.Log("table is not auto opened")
		wg.Done()
	}

	table, err := manager.CreateTable(tableEngineOption, tableEngineCallbacks, tableSetting)
	assert.Nil(t, err, "create table failed")

	// get table engine
	tableEngine, err = manager.GetTableEngine(table.ID)
	assert.Nil(t, err, "get table engine failed")

	// Initializing bot
	redeemChips := int64(30000)
	seat := -1
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Chuck", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Fred", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Loz", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Kimi", RedeemChips: redeemChips, Seat: seat},
	}

	// Preparing actors
	for _, p := range players {

		// Create new actor
		a := NewActor()

		// Initializing table engine adapter to communicate with table engine
		tc := NewTableEngineAdapter(tableEngine, table)
		a.SetAdapter(tc)

		// Initializing bot runner
		bot := NewBotRunner(p.PlayerID)
		bot.OnTableAutoJoinActionRequested(func(competitionID, tableID, playerID string) {
			assert.Nil(t, tableEngine.PlayerJoin(playerID), fmt.Sprintf("%s join error", playerID))
		})
		a.SetRunner(bot)

		actors = append(actors, a)
	}
	wg.Add(1)

	// Add players to table
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

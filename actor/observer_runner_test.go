package actor

import (
	"fmt"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	pokertable "github.com/weedbox/pokertable"
)

func TestActor_ObserverRunner_PlayerAct(t *testing.T) {
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
			MinChipUnit:         10,
			ActionTime:          10,
		},
	}
	tableUpdatedCallBack := func(table *pokertable.Table) {
		// Update table state via adapter
		for _, a := range actors {
			a.GetTable().UpdateTableState(table)
		}
	}
	tableErrorUpdatedCallBack := func(table *pokertable.Table, err error) {
		t.Log("[Table] Error:", err)
	}
	tableStateUpdatedCallBack := func(event string, table *pokertable.Table) {}
	tablePlayerStateUpdatedCallBack := func(string, string, *pokertable.TablePlayerState) {}
	table, err := manager.CreateTable(nil, tableSetting, tableUpdatedCallBack, tableErrorUpdatedCallBack, tableStateUpdatedCallBack, tablePlayerStateUpdatedCallBack)
	assert.Nil(t, err, "create table failed")

	// get table engine
	tableEngine, err = manager.GetTableEngine(table.ID)
	assert.Nil(t, err, "get table engine failed")

	// Initializing observer
	a := NewActor()

	tc := NewTableEngineAdapter(tableEngine, table)
	a.SetAdapter(tc)

	observer := NewObserverRunner()
	observer.OnTableStateUpdated(func(table *pokertable.Table) {
		if table.State.Status == pokertable.TableStateStatus_TableGameSettled {
			if table.State.GameState.Status.CurrentEvent == "GameClosed" {
				t.Log("GameClosed", table.State.GameState.GameID)

				if len(table.AlivePlayers()) == 1 {
					wg.Done()
					return
				}
			}
		}

		if table.State.Status == pokertable.TableStateStatus_TableGamePlaying {
			gs := table.State.GameState

			if gs.Status.LastAction == nil {
				return
			}

			if gs.Status.LastAction.Type == "big_blind" {
				//json, _ := table.GetJSON()
				//t.Log(json)

			}

			//t.Log(gs.Status.LastAction.Type, gs.Status.LastAction.Source, gs.Status.LastAction.Value)
		}
	})
	a.SetRunner(observer)

	wg.Add(1)

	actors = append(actors, a)

	// Initializing bot
	redeemChips := int64(3000)
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: redeemChips},
		{PlayerID: "Chuck", RedeemChips: redeemChips},
		{PlayerID: "Fred", RedeemChips: redeemChips},
	}

	// Preparing players
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

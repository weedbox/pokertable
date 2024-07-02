package actor

import (
	"fmt"
	"sync"
	"testing"
	"time"

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
			MinChipUnit:         9,
			ActionTime:          10,
		},
		Blind: pokertable.TableBlindState{
			Level:  1,
			Ante:   0,
			Dealer: 0,
			SB:     10,
			BB:     20,
		},
	}
	tableEngineOption := pokertable.NewTableEngineOptions()
	tableEngineOption.Interval = 1
	tableEngineCallbacks := pokertable.NewTableEngineCallbacks()
	tableEngineCallbacks.OnTableUpdated = func(table *pokertable.Table) {
		// Update table state via adapter
		for _, a := range actors {
			a.GetTable().UpdateTableState(table)
		}
	}
	tableEngineCallbacks.OnTableErrorUpdated = func(table *pokertable.Table, err error) {
		t.Log("[Table] Error:", err)
	}
	tableEngineCallbacks.OnAutoGameOpenEnd = func(competitionID, tableID string) {
		t.Log("AutoGameOpenEnd")
		wg.Done()
	}
	table, err := manager.CreateTable(tableEngineOption, tableEngineCallbacks, tableSetting)
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

			// if gs.Status.LastAction.Type == "big_blind" {
			// 	json, _ := table.GetJSON()
			// 	t.Log(json)

			// }

			//t.Log(gs.Status.LastAction.Type, gs.Status.LastAction.Source, gs.Status.LastAction.Value)
		}
	})
	a.SetRunner(observer)

	wg.Add(1)

	actors = append(actors, a)

	// Initializing bot
	redeemChips := int64(3000)
	seat := -1
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Chuck", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Fred", RedeemChips: redeemChips, Seat: seat},
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
		bot.OnTableAutoJoinActionRequested(func(competitionID, tableID, playerID string) {
			assert.Nil(t, tableEngine.PlayerJoin(playerID), fmt.Sprintf("%s join error", playerID))
		})
		a.SetRunner(bot)

		actors = append(actors, a)
	}

	// Add player to table
	for _, p := range players {
		assert.Nil(t, tableEngine.PlayerReserve(p), fmt.Sprintf("%s reserve error", p.PlayerID))

		go func(player pokertable.JoinPlayer) {
			time.Sleep(time.Microsecond * 10)
			assert.Nil(t, tableEngine.PlayerJoin(player.PlayerID), fmt.Sprintf("%s join error", player.PlayerID))
		}(p)
	}

	// Start game
	time.Sleep(time.Microsecond * 100)
	err = tableEngine.StartTableGame()
	assert.Nil(t, err)

	wg.Wait()
}

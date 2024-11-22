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

func TestActor_BotRunner_Humanize(t *testing.T) {
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
			ActionTime:          2,
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
	tableEngineOption.GameContinueInterval = 1
	tableEngineOption.OpenGameTimeout = 2
	tableEngineCallbacks := pokertable.NewTableEngineCallbacks()
	tableEngineCallbacks.OnTableUpdated = func(table *pokertable.Table) {
		// Update table state via adapter
		for _, a := range actors {
			a.GetTable().UpdateTableState(table)
		}

		if table.State.Status == pokertable.TableStateStatus_TableGameSettled {
			if table.State.GameState.Status.CurrentEvent == "GameClosed" {
				t.Log("GameClosed", table.State.GameState.GameID)

				if len(table.AlivePlayers()) == 1 {
					tableEngine.CloseTable()
					t.Log("Table closed")
					wg.Done()
					return
				}
			}
		}
	}
	tableEngineCallbacks.OnTableErrorUpdated = func(table *pokertable.Table, err error) {
		t.Log("[Table] Error:", err)
	}
	tableEngineCallbacks.OnAutoGameOpenEnd = func(competitionID, tableID string) {
		t.Log("AutoGameOpenEnd")
		wg.Done()
	}
	tableEngineCallbacks.OnReadyOpenFirstTableGame = func(competitionID, tableID string, gameCount int, players []*pokertable.TablePlayerState) {
		participants := map[string]int{}
		for idx, p := range players {
			participants[p.PlayerID] = idx
		}
		tableEngine.SetUpTableGame(gameCount, participants)
	}
	table, err := manager.CreateTable(tableEngineOption, tableEngineCallbacks, tableSetting)
	assert.Nil(t, err, "create table failed")

	// get table engine
	tableEngine, err = manager.GetTableEngine(table.ID)
	assert.Nil(t, err, "get table engine failed")

	// Initializing bot
	redeemChips := int64(3000)
	seat := -1
	players := []pokertable.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Chuck", RedeemChips: redeemChips, Seat: seat},
		{PlayerID: "Fred", RedeemChips: redeemChips, Seat: seat},
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
		bot.Humanized(true)
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

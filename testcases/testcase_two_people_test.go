package testcases

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokertable"
)

func TestTableGame_Two_People(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	// given conditions
	playerIDs := []string{"Fred", "Jeffrey"}
	redeemChips := int64(15000)
	players := funk.Map(playerIDs, func(playerID string) pokertable.JoinPlayer {
		return pokertable.JoinPlayer{
			PlayerID:    playerID,
			RedeemChips: redeemChips,
		}
	}).([]pokertable.JoinPlayer)

	// create manager & table
	manager := pokertable.NewManager()
	table, err := manager.CreateTable(NewDefaultTableSetting())
	assert.Nil(t, err, "create table failed")

	// get table engine
	tableEngine, err := manager.GetTableEngine(table.ID)
	assert.Nil(t, err, "get table engine failed")
	tableEngine.OnTableUpdated(func(table *pokertable.Table) {
		switch table.State.Status {
		case pokertable.TableStateStatus_TableGameOpened:
			DebugPrintTableGameOpened(*table)
		case pokertable.TableStateStatus_TableGamePlaying:
			t.Logf("[%s] %s:", table.State.GameState.Status.Round, table.State.GameState.Status.CurrentEvent)
			event, ok := pokerface.GameEventBySymbol[table.State.GameState.Status.CurrentEvent]
			if !ok {
				return
			}

			switch event {
			case pokerface.GameEvent_ReadyRequested:
				for _, playerID := range playerIDs {
					assert.Nil(t, tableEngine.PlayerReady(playerID), fmt.Sprintf("%s ready error", playerID))
					t.Logf(fmt.Sprintf("%s ready", playerID))
				}
			case pokerface.GameEvent_AnteRequested:
				for _, playerID := range playerIDs {
					ante := table.State.BlindState.Ante
					assert.Nil(t, tableEngine.PlayerPay(playerID, ante), fmt.Sprintf("%s pay ante error", playerID))
					t.Logf(fmt.Sprintf("%s pay ante %d", playerID, ante))
				}
			case pokerface.GameEvent_BlindsRequested:
				blind := table.State.BlindState

				// pay sb
				sbPlayerID := findPlayerID(table, "sb")
				assert.Nil(t, tableEngine.PlayerPay(sbPlayerID, blind.SB), fmt.Sprintf("%s pay sb error", sbPlayerID))
				t.Logf(fmt.Sprintf("%s pay sb %d", sbPlayerID, blind.SB))

				// pay bb
				bbPlayerID := findPlayerID(table, "bb")
				assert.Nil(t, tableEngine.PlayerPay(bbPlayerID, blind.BB), fmt.Sprintf("%s pay bb error", bbPlayerID))
				t.Logf(fmt.Sprintf("%s pay bb %d", bbPlayerID, blind.BB))
			case pokerface.GameEvent_RoundStarted:
				chips := int64(10)
				playerID, actions := currentPlayerMove(table)
				if funk.Contains(actions, "bet") {
					t.Logf(fmt.Sprintf("%s's move: bet %d", playerID, chips))
					assert.Nil(t, tableEngine.PlayerBet(playerID, chips), fmt.Sprintf("%s bet %d error", playerID, chips))
				} else if funk.Contains(actions, "check") {
					t.Logf(fmt.Sprintf("%s's move: check", playerID))
					assert.Nil(t, tableEngine.PlayerCheck(playerID), fmt.Sprintf("%s check error", playerID))
				} else if funk.Contains(actions, "call") {
					t.Logf(fmt.Sprintf("%s's move: call", playerID))
					assert.Nil(t, tableEngine.PlayerCall(playerID), fmt.Sprintf("%s call error", playerID))
				}
			}
		case pokertable.TableStateStatus_TableGameSettled:
			// check results
			assert.NotNil(t, table.State.GameState.Result, "invalid game result")
			assert.Equal(t, 1, table.State.GameCount)
			for _, playerResult := range table.State.GameState.Result.Players {
				playerIdx := table.State.GamePlayerIndexes[playerResult.Idx]
				player := table.State.PlayerStates[playerIdx]
				assert.Equal(t, playerResult.Final, player.Bankroll)
			}

			DebugPrintTableGameSettled(*table)

			wg.Done()
		}
	})

	// players buy in
	for _, joinPlayer := range players {
		assert.Nil(t, tableEngine.PlayerReserve(joinPlayer), fmt.Sprintf("%s reserve error", joinPlayer.PlayerID))
		assert.Nil(t, tableEngine.PlayerJoin(joinPlayer.PlayerID), fmt.Sprintf("%s join error", joinPlayer.PlayerID))
	}

	// start game
	tableEngine.UpdateBlind(1, 0, 0, 10, 20)
	assert.Nil(t, tableEngine.StartTableGame(), "start table game failed")

	wg.Wait()
}

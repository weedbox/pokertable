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

func TestTableGame_Preflop_Walk(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	// given conditions
	playerIDs := []string{"Fred", "Jeffrey", "Chuck"}
	redeemChips := int64(15000)
	players := funk.Map(playerIDs, func(playerID string) pokertable.JoinPlayer {
		return pokertable.JoinPlayer{
			PlayerID:    playerID,
			RedeemChips: redeemChips,
		}
	}).([]pokertable.JoinPlayer)

	// create table engine
	tableEngine := pokertable.NewTableEngine()
	tableEngine.OnTableUpdated(func(table *pokertable.Table) {
		if table == nil {
			return
		}

		switch table.State.Status {
		case pokertable.TableStateStatus_TableGameOpened:
			DebugPrintTableGameOpened(*table)
		case pokertable.TableStateStatus_TableGamePlaying:
			t.Logf("[%s] %s:", table.State.GameState.Status.Round, table.State.GameState.Status.CurrentEvent)
			switch table.State.GameState.Status.CurrentEvent {
			case gameEvent(pokerface.GameEvent_ReadyRequested):
				for _, playerID := range playerIDs {
					assert.Nil(t, tableEngine.PlayerReady(table.ID, playerID), fmt.Sprintf("%s ready error", playerID))
					t.Logf(fmt.Sprintf("%s ready", playerID))
				}
			case gameEvent(pokerface.GameEvent_AnteRequested):
				for _, playerID := range playerIDs {
					ante := table.State.BlindState.CurrentBlindLevel().BlindLevel.Ante
					assert.Nil(t, tableEngine.PlayerPay(table.ID, playerID, ante), fmt.Sprintf("%s pay ante error", playerID))
					t.Logf(fmt.Sprintf("%s pay ante %d", playerID, ante))
				}
			case gameEvent(pokerface.GameEvent_BlindsRequested):
				blind := table.State.BlindState.CurrentBlindLevel().BlindLevel

				// pay sb
				sbPlayerID := findPlayerID(table, "sb")
				assert.Nil(t, tableEngine.PlayerPay(table.ID, sbPlayerID, blind.SB), fmt.Sprintf("%s pay sb error", sbPlayerID))
				t.Logf(fmt.Sprintf("%s pay sb %d", sbPlayerID, blind.SB))

				// pay bb
				bbPlayerID := findPlayerID(table, "bb")
				assert.Nil(t, tableEngine.PlayerPay(table.ID, bbPlayerID, blind.BB), fmt.Sprintf("%s pay bb error", bbPlayerID))
				t.Logf(fmt.Sprintf("%s pay bb %d", bbPlayerID, blind.BB))
			case gameEvent(pokerface.GameEvent_RoundStarted):
				playerID, actions := currentPlayerMove(table)
				if funk.Contains(actions, "fold") {
					t.Logf(fmt.Sprintf("%s's move: fold", playerID))
					assert.Nil(t, tableEngine.PlayerFold(table.ID, playerID), fmt.Sprintf("%s fold error", playerID))
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

	// create a new table
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err, "create table failed")

	// players buy in
	for _, joinPlayer := range players {
		assert.Nil(t, tableEngine.PlayerReserve(table.ID, joinPlayer), fmt.Sprintf("%s reserve error", joinPlayer.PlayerID))
		assert.Nil(t, tableEngine.PlayerJoin(table.ID, joinPlayer.PlayerID), fmt.Sprintf("%s join error", joinPlayer.PlayerID))
	}

	// start game
	assert.Nil(t, tableEngine.StartTableGame(table.ID), "start table game failed")

	wg.Wait()
}

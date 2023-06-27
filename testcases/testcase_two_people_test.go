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
				chips := int64(10)
				playerID, actions := currentPlayerMove(table)
				if funk.Contains(actions, "bet") {
					t.Logf(fmt.Sprintf("%s's move: bet %d", playerID, chips))
					assert.Nil(t, tableEngine.PlayerBet(table.ID, playerID, chips), fmt.Sprintf("%s bet %d error", playerID, chips))
				} else if funk.Contains(actions, "check") {
					t.Logf(fmt.Sprintf("%s's move: check", playerID))
					assert.Nil(t, tableEngine.PlayerCheck(table.ID, playerID), fmt.Sprintf("%s check error", playerID))
				} else if funk.Contains(actions, "call") {
					t.Logf(fmt.Sprintf("%s's move: call", playerID))
					assert.Nil(t, tableEngine.PlayerCall(table.ID, playerID), fmt.Sprintf("%s call error", playerID))
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
		assert.Nil(t, tableEngine.PlayerJoin(table.ID, joinPlayer), fmt.Sprintf("%s buy in error", joinPlayer.PlayerID))
	}

	// start game
	assert.Nil(t, tableEngine.StartTableGame(table.ID), "start table game failed")

	wg.Wait()

	// // given conditions
	// playerIDs := []string{"Fred", "Jeffrey"}
	// playersAutoPlayActions := func(tableEngine pokertable.TableEngine, tableID string) {
	// 	// game started
	// 	// all players ready
	// 	table, err := tableEngine.GetTable(tableID)
	// 	assert.Nil(t, err, "get table failed")
	// 	AllGamePlayersReady(t, tableEngine, table)

	// 	// preflop
	// 	// pay sb
	// 	sb := table.State.BlindState.CurrentBlindLevel().BlindLevel.SB
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay sb %d", sb))
	// 	err = tableEngine.PlayerPay(tableID, FindCurrentPlayerID(table), sb)
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay sb %d", sb), err))
	// 	fmt.Printf("[PlayerPaySB] dealer receive sb %d.\n", sb)

	// 	// pay bb
	// 	bb := table.State.BlindState.CurrentBlindLevel().BlindLevel.BB
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay bb %d", bb))
	// 	err = tableEngine.PlayerPay(tableID, FindCurrentPlayerID(table), bb)
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), fmt.Sprintf("pay bb %d", sb), err))
	// 	fmt.Printf("[PlayerPaySB] dealer receive bb %d.\n", bb)

	// 	// all players ready
	// 	AllGamePlayersReady(t, tableEngine, table)

	// 	// dealer/sb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	// 	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// 	// bb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "check")
	// 	err = tableEngine.PlayerCheck(tableID, FindCurrentPlayerID(table))
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "check", err))

	// 	// flop
	// 	// all players ready
	// 	AllGamePlayersReady(t, tableEngine, table)

	// 	// dealer/sb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	// 	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// 	// bb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	// 	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// 	// turn
	// 	// all players ready
	// 	AllGamePlayersReady(t, tableEngine, table)

	// 	// dealer/sb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	// 	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// 	// bb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	// 	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))

	// 	// river
	// 	// all players ready
	// 	AllGamePlayersReady(t, tableEngine, table)

	// 	// dealer/sb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "bet 10")
	// 	err = tableEngine.PlayerBet(tableID, FindCurrentPlayerID(table), 10)
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "bet 10", err))

	// 	// bb move
	// 	PrintPlayerActionLog(table, FindCurrentPlayerID(table), "call")
	// 	err = tableEngine.PlayerCall(tableID, FindCurrentPlayerID(table))
	// 	assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "call", err))
	// }

	// // create a table
	// tableEngine := pokertable.NewTableEngine()
	// tableEngine.OnTableUpdated(func(table *pokertable.Table) {
	// 	switch table.State.Status {
	// 	case pokertable.TableStateStatus_TableGameOpened:
	// 		DebugPrintTableGameOpened(*table)
	// 	case pokertable.TableStateStatus_TableGameSettled:
	// 		DebugPrintTableGameSettled(*table)
	// 	}
	// })
	// tableSetting := NewDefaultTableSetting()
	// table, err := tableEngine.CreateTable(tableSetting)
	// assert.Nil(t, err)

	// // buy in
	// redeemChips := int64(15000)
	// players := make([]pokertable.JoinPlayer, 0)
	// for _, playerID := range playerIDs {
	// 	players = append(players, pokertable.JoinPlayer{
	// 		PlayerID:    playerID,
	// 		RedeemChips: redeemChips,
	// 	})
	// }
	// for _, joinPlayer := range players {
	// 	err = tableEngine.PlayerJoin(table.ID, joinPlayer)
	// 	assert.Nil(t, err)
	// }

	// // start game
	// err = tableEngine.StartTableGame(table.ID)
	// assert.Nil(t, err)

	// playersAutoPlayActions(tableEngine, table.ID)
}

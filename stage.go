package pokertable

import (
	"fmt"
	"time"
)

// TableInit 初始化桌
func (engine *tableEngine) TableInit(table Table) Table {
	// update StartGameAt
	table.State.StartGameAt = time.Now().Unix()

	// activate blind levels
	newBlindState := engine.ActivateBlindState(table.State.StartGameAt, *table.State.BlindState)
	table.State.BlindState = &newBlindState

	return table
}

// GameOpen 開始本手遊戲
func (engine *tableEngine) GameOpen(table Table) (Table, error) {
	// Step 1: 重設桌次狀態
	table.Reset()

	// Step 2: 檢查參賽資格
	for i := 0; i < len(table.State.PlayerStates); i++ {
		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if table.State.PlayerStates[i].IsParticipated || table.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		// 檢查後手 (有錢的玩家可參賽)
		table.State.PlayerStates[i].IsParticipated = table.State.PlayerStates[i].Bankroll > 0
	}

	// Step 3: 處理可參賽玩家剩餘一人時，桌上有其他玩家情形
	if len(table.ParticipatedPlayers()) < 2 {
		for i := 0; i < len(table.State.PlayerStates); i++ {
			if table.State.PlayerStates[i].Bankroll == 0 {
				continue
			}

			table.State.PlayerStates[i].IsParticipated = true
			table.State.PlayerStates[i].IsBetweenDealerBB = false
		}
	}

	// Step 4: 計算新 Dealer SeatIndex & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(table.State.GameCount, table.State.CurrentDealerSeatIndex, table.Meta.CompetitionMeta.TableMinPlayingCount, table.Meta.CompetitionMeta.TableMaxSeatCount, table.State.PlayerStates, table.State.PlayerSeatMap)
	newDealerTableSeatIdx := table.State.PlayerStates[newDealerPlayerIdx].SeatIndex

	// Step 5: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(table.State.PlayerStates); i++ {
		if !table.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if newDealerTableSeatIdx-table.State.CurrentDealerSeatIndex < 0 {
			for j := table.State.CurrentDealerSeatIndex + 1; j < newDealerTableSeatIdx+table.Meta.CompetitionMeta.TableMaxSeatCount; j++ {
				if (j % table.Meta.CompetitionMeta.TableMaxSeatCount) != table.State.PlayerStates[i].SeatIndex {
					continue
				}

				table.State.PlayerStates[i].IsParticipated = true
				table.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		} else {
			for j := table.State.CurrentDealerSeatIndex + 1; j < newDealerTableSeatIdx; j++ {
				if j != table.State.PlayerStates[i].SeatIndex {
					continue
				}

				table.State.PlayerStates[i].IsParticipated = true
				table.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		}
	}

	// Step 6: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	playingPlayerIndexes := FindPlayingPlayerIndexes(newDealerTableSeatIdx, table.State.PlayerSeatMap, table.State.PlayerStates)
	table.State.PlayingPlayerIndexes = playingPlayerIndexes

	// Step 7: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(table.Meta.CompetitionMeta.Rule, table.State.PlayerStates, playingPlayerIndexes)
	for playerIdx := 0; playerIdx < len(table.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && table.State.PlayerStates[playerIdx].IsParticipated {
			table.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 8: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置)
	table.State.GameCount = table.State.GameCount + 1
	table.State.CurrentDealerSeatIndex = newDealerTableSeatIdx
	if len(playingPlayerIndexes) == 2 {
		bbPlayerIdx := playingPlayerIndexes[1]
		table.State.CurrentBBSeatIndex = table.State.PlayerStates[bbPlayerIdx].SeatIndex
	} else if len(playingPlayerIndexes) > 2 {
		bbPlayerIdx := playingPlayerIndexes[2]
		table.State.CurrentBBSeatIndex = table.State.PlayerStates[bbPlayerIdx].SeatIndex
	} else {
		table.State.CurrentBBSeatIndex = UnsetValue
	}

	// Step 9: 更新當前桌次事件
	table.State.Status = TableStateStatus_TableGameMatchOpen

	// Step 10: 啟動本手遊戲引擎 & 更新遊戲狀態
	blind := *table.State.BlindState.LevelStates[table.State.BlindState.CurrentLevelIndex]
	dealerBlindTimes := table.Meta.CompetitionMeta.Blind.DealerBlindTimes
	gameEngineSetting := engine.newGameEngineSetting(table.Meta.CompetitionMeta.Rule, blind, dealerBlindTimes, table.State.PlayerStates, table.State.PlayingPlayerIndexes)
	gameState, err := engine.gameEngine.Start(gameEngineSetting)
	table.State.GameState = gameState

	engine.debugPrintTable(fmt.Sprintf("第 (%d) 手開局資訊", table.State.GameCount), table) // TODO: test only, remove it later on
	return table, err
}

// TableSettlement 本手遊戲結算
func (engine *tableEngine) TableSettlement(table Table) Table {
	// Step 1: 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range table.State.GameState.Result.Players {
		playerIdx := table.State.PlayingPlayerIndexes[player.Idx]
		table.State.PlayerStates[playerIdx].Bankroll = player.Final
	}

	// Step 2: 更新盲注 Level
	table.State.BlindState.Update()

	// Step 3: 依照桌次目前狀況更新事件
	if !table.State.BlindState.IsFinalBuyInLevel() && len(table.AlivePlayers()) < 2 {
		table.State.Status = TableStateStatus_TableGamePaused
	} else if table.State.BlindState.IsBreaking() {
		table.State.Status = TableStateStatus_TableGamePaused
	} else if engine.isTableClose(table.EndGameAt(), table.AlivePlayers(), table.State.BlindState.IsFinalBuyInLevel()) {
		table.State.Status = TableStateStatus_TableGameClosed
	}

	return table
}

func (engine *tableEngine) ActivateBlindState(startGameAt int64, blindState TableBlindState) TableBlindState {
	for idx, levelState := range blindState.LevelStates {
		if levelState.Level == blindState.InitialLevel {
			blindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := startGameAt
	for i := (blindState.InitialLevel - 1); i < len(blindState.LevelStates); i++ {
		if i == blindState.InitialLevel-1 {
			blindState.LevelStates[i].LevelEndAt = blindStartAt
		} else {
			blindState.LevelStates[i].LevelEndAt = blindState.LevelStates[i-1].LevelEndAt
		}
		blindPassedSeconds := int64((time.Duration(blindState.LevelStates[i].DurationMins) * time.Minute).Seconds())
		blindState.LevelStates[i].LevelEndAt += blindPassedSeconds
	}
	return blindState
}

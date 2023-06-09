package pokertable

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

func (te *tableEngine) CreateTable(tableSetting TableSetting) (Table, error) {
	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.CompetitionMeta.TableMaxSeatCount {
		return Table{}, ErrInvalidCreateTableSetting
	}

	meta := TableMeta{
		ShortID:         tableSetting.ShortID,
		Code:            tableSetting.Code,
		Name:            tableSetting.Name,
		InvitationCode:  tableSetting.InvitationCode,
		CompetitionMeta: tableSetting.CompetitionMeta,
	}

	finalBuyInLevelIdx := UnsetValue
	if tableSetting.CompetitionMeta.Blind.FinalBuyInLevel != UnsetValue {
		for idx, blindLevel := range tableSetting.CompetitionMeta.Blind.Levels {
			if blindLevel.Level == tableSetting.CompetitionMeta.Blind.FinalBuyInLevel {
				finalBuyInLevelIdx = idx
				break
			}
		}
	}

	blindState := TableBlindState{
		FinalBuyInLevelIndex: finalBuyInLevelIdx,
		InitialLevel:         tableSetting.CompetitionMeta.Blind.InitialLevel,
		CurrentLevelIndex:    UnsetValue,
		LevelStates: funk.Map(tableSetting.CompetitionMeta.Blind.Levels, func(blindLevel BlindLevel) *TableBlindLevelState {
			return &TableBlindLevelState{
				Level:        blindLevel.Level,
				SBChips:      blindLevel.SBChips,
				BBChips:      blindLevel.BBChips,
				AnteChips:    blindLevel.AnteChips,
				DurationMins: blindLevel.DurationMins,
				LevelEndAt:   UnsetValue,
			}
		}).([]*TableBlindLevelState),
	}

	state := TableState{
		GameCount:              0,
		StartGameAt:            UnsetValue,
		BlindState:             &blindState,
		CurrentDealerSeatIndex: UnsetValue,
		CurrentBBSeatIndex:     UnsetValue,
		PlayerSeatMap:          NewDefaultSeatMap(tableSetting.CompetitionMeta.TableMaxSeatCount),
		PlayerStates:           make([]*TablePlayerState, 0),
		PlayingPlayerIndexes:   make([]int, 0),
		Status:                 TableStateStatus_TableGameCreated,
	}

	// handle auto join players
	if len(tableSetting.JoinPlayers) > 0 {
		// auto join players
		state.PlayerStates = funk.Map(tableSetting.JoinPlayers, func(p JoinPlayer) *TablePlayerState {
			return &TablePlayerState{
				PlayerID:          p.PlayerID,
				SeatIndex:         UnsetValue,
				Positions:         []string{Position_Unknown},
				IsParticipated:    true,
				IsBetweenDealerBB: false,
				Bankroll:          p.RedeemChips,
			}
		}).([]*TablePlayerState)

		// update seats
		for playerIdx := 0; playerIdx < len(state.PlayerStates); playerIdx++ {
			seatIdx := RandomSeatIndex(state.PlayerSeatMap)
			state.PlayerSeatMap[seatIdx] = playerIdx
			state.PlayerStates[playerIdx].SeatIndex = seatIdx
		}
	}

	return Table{
		ID:       uuid.New().String(),
		Meta:     meta,
		State:    &state,
		UpdateAt: time.Now().Unix(),
	}, nil
}

func (te *tableEngine) CloseTable(table Table, status TableStateStatus) Table {
	table.State.Status = status
	table.Update()
	return table
}

func (te *tableEngine) StartGame(table Table) (Table, error) {
	// 初始化桌 & 開局
	// update StartGameAt
	table.State.StartGameAt = time.Now().Unix()

	// activate blind levels
	for idx, levelState := range table.State.BlindState.LevelStates {
		if levelState.Level == table.State.BlindState.InitialLevel {
			table.State.BlindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := table.State.StartGameAt
	for i := (table.State.BlindState.InitialLevel - 1); i < len(table.State.BlindState.LevelStates); i++ {
		if i == table.State.BlindState.InitialLevel-1 {
			table.State.BlindState.LevelStates[i].LevelEndAt = blindStartAt
		} else {
			table.State.BlindState.LevelStates[i].LevelEndAt = table.State.BlindState.LevelStates[i-1].LevelEndAt
		}
		blindPassedSeconds := int64((time.Duration(table.State.BlindState.LevelStates[i].DurationMins) * time.Minute).Seconds())
		table.State.BlindState.LevelStates[i].LevelEndAt += blindPassedSeconds
	}

	return te.GameOpen(table)
}

// GameOpen 開始本手遊戲
func (te *tableEngine) GameOpen(table Table) (Table, error) {
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
	gameEngineSetting := te.newGameEngineSetting(table.Meta.CompetitionMeta.Rule, blind, dealerBlindTimes, table.State.PlayerStates, table.State.PlayingPlayerIndexes)
	gameState, err := te.gameEngine.Start(gameEngineSetting)
	table.State.GameState = gameState

	te.debugPrintTable(fmt.Sprintf("第 (%d) 手開局資訊", table.State.GameCount), table) // TODO: test only, remove it later on
	return table, err
}

// TableSettlement 本手遊戲結算
func (te *tableEngine) TableSettlement(table Table) Table {
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
	} else if te.isTableClose(table.EndGameAt(), table.AlivePlayers(), table.State.BlindState.IsFinalBuyInLevel()) {
		table.State.Status = TableStateStatus_TableGameClosed
	}

	return table
}

/*
	isTableClose 計算本桌是否已結束
	  - 結束條件 1: 達到賽局結束時間
	  - 結束條件 2: 停止買入後且存活玩家剩餘 1 人
*/
func (te *tableEngine) isTableClose(endGameAt int64, alivePlayers []*TablePlayerState, isFinalBuyInLevel bool) bool {
	return time.Now().Unix() > endGameAt || (isFinalBuyInLevel && len(alivePlayers) == 1)
}

func (te *tableEngine) getGameStateEventName(event pokerface.GameEvent) string {
	return pokerface.GameEventSymbols[event]
}

func (te *tableEngine) newGameEngineSetting(rule string, blind TableBlindLevelState, dealerBlindTimes int, players []*TablePlayerState, playingPlayerIndexes []int) GameEngineSetting {
	setting := GameEngineSetting{
		Rule: rule,
		Ante: blind.AnteChips,
		Blind: pokerface.BlindSetting{
			Dealer: blind.AnteChips * (int64(dealerBlindTimes) - 1),
			SB:     blind.SBChips,
			BB:     blind.BBChips,
		},
	}
	playerSettings := make([]*pokerface.PlayerSetting, 0)
	for _, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		playerSettings = append(playerSettings, &pokerface.PlayerSetting{
			Bankroll:  player.Bankroll,
			Positions: player.Positions,
		})
	}
	setting.Players = playerSettings
	return setting
}

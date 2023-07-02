package pokertable

import (
	"fmt"
	"sync"
	"time"

	"github.com/weedbox/pokerface"
)

func (te *tableEngine) validateGameMove(gamePlayerIdx int) error {
	// check table status
	if te.table.State.Status != TableStateStatus_TableGamePlaying {
		return ErrTablePlayerInvalidGameAction
	}

	// check game player index
	if gamePlayerIdx == UnsetValue {
		return ErrTablePlayerNotFound
	}

	return nil
}

func (te *tableEngine) delay(interval int, fn func() error) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(1)

	te.tb.NewTask(time.Duration(interval)*time.Second, func(isCancelled bool) {
		defer wg.Done()

		if isCancelled {
			return
		}

		err = fn()
	})

	wg.Wait()
	return err
}

func (te *tableEngine) emitEvent(eventName string, playerID string) {
	// refresh table
	te.table.UpdateAt = time.Now().Unix()
	te.table.UpdateSerial++

	// emit event
	fmt.Printf("->[Table %s][#%d][%d][%s] emit Event: %s\n", te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName)
	te.onTableUpdated(te.table)
}

func (te *tableEngine) emitErrorEvent(eventName string, playerID string, err error) {
	fmt.Printf("->[Table %s][#%d][%d][%s] emit ERROR Event: %s, Error: %v\n", te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName, err)
	te.onTableErrorUpdated(te.table, err)
}

func (te *tableEngine) updateGameState(gs *pokerface.GameState) {
	te.table.State.GameState = gs

	event, ok := pokerface.GameEventBySymbol[gs.Status.CurrentEvent]
	if !ok {
		te.emitErrorEvent("handle updateGameState", "", ErrGameUnknownEvent)
		return
	}

	switch event {
	case pokerface.GameEvent_GameClosed:
		if err := te.onGameClosed(); err != nil {
			te.emitErrorEvent("onGameClosed", "", err)
		}
	default:
		te.emitEvent(gs.Status.CurrentEvent, "")
	}
}

func (te *tableEngine) openGame() {
	// Step 1: 更新狀態
	te.table.State.Status = TableStateStatus_TableGameOpened

	// Step 2: 檢查參賽資格
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		// 沒有入桌玩家直接不參加
		if !te.table.State.PlayerStates[i].IsIn {
			te.table.State.PlayerStates[i].IsParticipated = false
			continue
		}

		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if te.table.State.PlayerStates[i].IsParticipated || te.table.State.PlayerStates[i].IsBetweenDealerBB {
			te.table.State.PlayerStates[i].IsParticipated = te.table.State.PlayerStates[i].Bankroll > 0
			continue
		}

		// 檢查後手 (有錢的玩家可參賽)
		te.table.State.PlayerStates[i].IsParticipated = te.table.State.PlayerStates[i].Bankroll > 0
	}

	// Step 3: 處理可參賽玩家剩餘一人時，桌上有其他玩家情形
	if len(te.table.ParticipatedPlayers()) < 2 {
		for i := 0; i < len(te.table.State.PlayerStates); i++ {
			// 沒入桌或沒籌碼玩家不能玩
			if te.table.State.PlayerStates[i].Bankroll == 0 || !te.table.State.PlayerStates[i].IsIn {
				continue
			}

			te.table.State.PlayerStates[i].IsParticipated = true
			te.table.State.PlayerStates[i].IsBetweenDealerBB = false
		}
	}

	// Step 4: 計算新 Dealer Seat & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(te.table.State.GameCount, te.table.State.CurrentDealerSeat, te.table.Meta.CompetitionMeta.TableMinPlayerCount, te.table.Meta.CompetitionMeta.TableMaxSeatCount, te.table.State.PlayerStates, te.table.State.SeatMap)
	newDealerTableSeatIdx := te.table.State.PlayerStates[newDealerPlayerIdx].Seat

	// Step 5: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		if !te.table.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if newDealerTableSeatIdx-te.table.State.CurrentDealerSeat < 0 {
			for j := te.table.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx+te.table.Meta.CompetitionMeta.TableMaxSeatCount; j++ {
				if (j % te.table.Meta.CompetitionMeta.TableMaxSeatCount) != te.table.State.PlayerStates[i].Seat {
					continue
				}

				if !te.table.State.PlayerStates[i].IsIn {
					continue
				}

				te.table.State.PlayerStates[i].IsParticipated = true
				te.table.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		} else {
			for j := te.table.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx; j++ {
				if j != te.table.State.PlayerStates[i].Seat {
					continue
				}

				if !te.table.State.PlayerStates[i].IsIn {
					continue
				}

				te.table.State.PlayerStates[i].IsParticipated = true
				te.table.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		}
	}

	// Step 6: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	gamePlayerIndexes := FindGamePlayerIndexes(newDealerTableSeatIdx, te.table.State.SeatMap, te.table.State.PlayerStates)
	te.table.State.GamePlayerIndexes = gamePlayerIndexes

	// Step 7: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(te.table.Meta.CompetitionMeta.Rule, te.table.State.PlayerStates, te.table.State.GamePlayerIndexes)
	for playerIdx := 0; playerIdx < len(te.table.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && te.table.State.PlayerStates[playerIdx].IsParticipated {
			te.table.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 8: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置)
	te.table.State.GameCount = te.table.State.GameCount + 1
	te.table.State.CurrentDealerSeat = newDealerTableSeatIdx
	if len(gamePlayerIndexes) == 2 {
		bbPlayerIdx := gamePlayerIndexes[1]
		te.table.State.CurrentBBSeat = te.table.State.PlayerStates[bbPlayerIdx].Seat
	} else if len(gamePlayerIndexes) > 2 {
		bbPlayerIdx := gamePlayerIndexes[2]
		te.table.State.CurrentBBSeat = te.table.State.PlayerStates[bbPlayerIdx].Seat
	} else {
		te.table.State.CurrentBBSeat = UnsetValue
	}
}

func (te *tableEngine) startGame() error {
	rule := te.table.Meta.CompetitionMeta.Rule
	blind := te.table.State.BlindState.LevelStates[te.table.State.BlindState.CurrentLevelIndex].BlindLevel
	DealerBlindTime := te.table.Meta.CompetitionMeta.Blind.DealerBlindTime

	// create game options
	opts := pokerface.NewStardardGameOptions()
	opts.Deck = pokerface.NewStandardDeckCards()

	if rule == CompetitionRule_ShortDeck {
		opts = pokerface.NewShortDeckGameOptions()
		opts.Deck = pokerface.NewShortDeckCards()
	} else if rule == CompetitionRule_Omaha {
		opts.HoleCardsCount = 4
		opts.RequiredHoleCardsCount = 2
	}

	// preparing blind
	dealer := int64(0)
	if DealerBlindTime > 0 {
		dealer = blind.Ante * (int64(DealerBlindTime) - 1)
	}

	opts.Ante = blind.Ante
	opts.Blind = pokerface.BlindSetting{
		Dealer: dealer,
		SB:     blind.SB,
		BB:     blind.BB,
	}

	// preparing players
	playerSettings := make([]*pokerface.PlayerSetting, 0)
	for _, playerIdx := range te.table.State.GamePlayerIndexes {
		player := te.table.State.PlayerStates[playerIdx]
		playerSettings = append(playerSettings, &pokerface.PlayerSetting{
			Bankroll:  player.Bankroll,
			Positions: player.Positions,
		})
	}
	opts.Players = playerSettings

	// create game
	te.game = NewGame(te.gameBackend, opts)
	te.game.OnGameStateUpdated(func(gs *pokerface.GameState) {
		te.updateGameState(gs)
	})
	te.game.OnGameErrorUpdated(func(gs *pokerface.GameState, err error) {
		te.table.State.GameState = gs
		go te.emitErrorEvent("OnGameErrorUpdated", "", err)
	})

	// start game
	if err := te.game.Start(); err != nil {
		return err
	}

	te.table.State.Status = TableStateStatus_TableGamePlaying
	return nil
}

func (te *tableEngine) settleGame() {
	te.table.State.Status = TableStateStatus_TableGameSettled

	// 更新現在盲注資訊
	now := time.Now().Unix()
	for idx, levelState := range te.table.State.BlindState.LevelStates {
		timeDiff := now - levelState.EndAt
		if timeDiff < 0 {
			te.table.State.BlindState.CurrentLevelIndex = idx
			break
		} else {
			if idx+1 < len(te.table.State.BlindState.LevelStates) {
				te.table.State.BlindState.CurrentLevelIndex = idx + 1
			}
		}
	}

	// 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range te.table.State.GameState.Result.Players {
		playerIdx := te.table.State.GamePlayerIndexes[player.Idx]
		te.table.State.PlayerStates[playerIdx].Bankroll = player.Final
	}
	te.emitEvent("SettleTableGameResult", "")
}

func (te *tableEngine) continueGame() error {
	// 檢查是否暫停
	if te.table.ShouldPause() {
		te.table.State.Status = TableStateStatus_TablePausing
	} else {
		te.table.State.Status = TableStateStatus_TableGameStandby
	}

	// reset state
	te.table.State.GamePlayerIndexes = []int{}
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		te.table.State.PlayerStates[i].Positions = make([]string, 0)
	}

	// reset ready group
	te.rg.Stop()
	te.rg.ResetParticipants()

	if te.table.State.Status == TableStateStatus_TablePausing && te.table.State.BlindState.IsBreaking() {
		// resume game from breaking
		endAt := te.table.State.BlindState.LevelStates[te.table.State.BlindState.CurrentLevelIndex].EndAt
		if err := te.tb.NewTaskWithDeadline(time.Unix(endAt, 0), func(isCancelled bool) {
			if isCancelled {
				return
			}

			if te.table.State.Status != TableStateStatus_TableBalancing && len(te.table.AlivePlayers()) >= 2 {
				if err := te.TableGameOpen(); err != nil {
					te.emitErrorEvent("resume game from breaking", "", err)
				}
			}
		}); err != nil {
			return err
		}
	} else if te.table.State.Status == TableStateStatus_TableGameStandby && te.table.Meta.CompetitionMeta.Mode == CompetitionMode_CT {
		if err := te.delay(te.options.Interval, func() error {
			// 自動開桌條件: 非 TableStateStatus_TableGamePlaying 或 非 TableStateStatus_TableBalancing
			stopOpen := te.table.State.Status == TableStateStatus_TableGamePlaying || te.table.State.Status == TableStateStatus_TableBalancing
			if !stopOpen && len(te.table.AlivePlayers()) >= 2 {
				return te.TableGameOpen()
			}
			return nil
		}); err != nil {
			return err
		}
	}
	te.emitEvent("ContinueGame", "")
	return nil
}

func (te *tableEngine) onGameClosed() error {
	te.settleGame()
	return te.continueGame()
}

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
		te.emitTableStateEvent(TableStateEvent_GameUpdated)
	}
}

func (te *tableEngine) openGame() error {
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
	if len(te.table.ParticipatedPlayers()) < te.table.Meta.TableMinPlayerCount {
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
	newDealerPlayerIdx := FindDealerPlayerIndex(te.table.State.GameCount, te.table.State.CurrentDealerSeat, te.table.Meta.TableMinPlayerCount, te.table.Meta.TableMaxSeatCount, te.table.State.PlayerStates, te.table.State.SeatMap)
	newDealerTableSeatIdx := te.table.State.PlayerStates[newDealerPlayerIdx].Seat

	// Step 5: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		if !te.table.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if newDealerTableSeatIdx-te.table.State.CurrentDealerSeat < 0 {
			for j := te.table.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx+te.table.Meta.TableMaxSeatCount; j++ {
				if (j % te.table.Meta.TableMaxSeatCount) != te.table.State.PlayerStates[i].Seat {
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
	if len(gamePlayerIndexes) < te.table.Meta.TableMinPlayerCount {
		return ErrTableOpenGameFailed
	}
	te.table.State.GamePlayerIndexes = gamePlayerIndexes

	// Step 7: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(te.table.Meta.Rule, te.table.State.PlayerStates, te.table.State.GamePlayerIndexes)
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

	return nil
}

func (te *tableEngine) startGame() error {
	rule := te.table.Meta.Rule
	blind := te.table.State.BlindState

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
	opts.Ante = blind.Ante
	opts.Blind = pokerface.BlindSetting{
		Dealer: blind.Dealer,
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

	// 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range te.table.State.GameState.Result.Players {
		playerIdx := te.table.State.GamePlayerIndexes[player.Idx]
		te.table.State.PlayerStates[playerIdx].Bankroll = player.Final
	}
	te.emitEvent("SettleTableGameResult", "")
	te.emitTableStateEvent(TableStateEvent_GameSettled)
}

func (te *tableEngine) continueGame() error {
	// reset state
	te.table.State.GamePlayerIndexes = []int{}
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		te.table.State.PlayerStates[i].Positions = make([]string, 0)
	}

	// 檢查是否暫停
	if te.table.ShouldPause() {
		// 暫停處理
		te.table.State.Status = TableStateStatus_TablePausing
		te.emitEvent("ContinueGame -> Pause", "")
		te.emitTableStateEvent(TableStateEvent_StatusUpdated)
	} else {
		// 正常繼續新的一手
		te.table.State.Status = TableStateStatus_TableGameStandby
		te.emitEvent("ContinueGame -> Standby", "")
		te.emitTableStateEvent(TableStateEvent_StatusUpdated)

		if err := te.delay(te.options.Interval, func() error {
			// 自動開下一手條件: 非 TableStateStatus_TableGamePlaying 或 非 TableStateStatus_TableBalancing 或 非 TableStateStatus_TableBalancing 且有籌碼玩家 >= 最小開打人數
			stopOpen := (te.table.State.Status == TableStateStatus_TableGamePlaying || te.table.State.Status == TableStateStatus_TableBalancing || te.table.State.Status == TableStateStatus_TableClosed) && len(te.table.AlivePlayers()) >= te.table.Meta.TableMinPlayerCount
			fmt.Printf("table [%s] stop open: %v\n", te.table.ID, stopOpen)
			if !stopOpen {
				return te.TableGameOpen()
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func (te *tableEngine) onGameClosed() error {
	te.settleGame()
	return te.continueGame()
}

package pokertable

import (
	"fmt"
	"time"

	"github.com/weedbox/syncsaga"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokerface/settlement"
)

func (te *tableEngine) openGame(oldTable *Table) (*Table, error) {
	// Step 1: Check TableState
	if !oldTable.State.BlindState.IsSet() {
		return oldTable, ErrTableOpenGameFailed
	}

	if oldTable.State.BlindState.IsBreaking() {
		return oldTable, ErrTableOpenGameFailedInBlindBreakingLevel
	}

	// Step 2: Clone Table for calculation
	cloneTable, err := oldTable.Clone()
	if err != nil {
		return oldTable, err
	}

	// Step 3: 更新狀態
	cloneTable.State.Status = TableStateStatus_TableGameOpened

	// Step 4: 計算座位
	if !te.sm.IsInitPositions() {
		if err := te.sm.InitPositions(true); err != nil {
			return oldTable, ErrTableOpenGameFailed
		}
	} else {
		if err := te.sm.RotatePositions(); err != nil {
			return oldTable, ErrTableOpenGameFailed
		}
	}

	// Step 5: 更新參與本手的玩家資訊
	// update player is_participated
	for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
		player := cloneTable.State.PlayerStates[i]
		active, err := te.sm.IsPlayerActive(player.PlayerID)
		if err != nil {
			return oldTable, err
		}
		player.IsParticipated = active
	}

	// update gamePlayerIndexes & positions
	cloneTable.State.GamePlayerIndexes = te.calcGamePlayerIndexes(
		cloneTable.Meta.Rule,
		cloneTable.Meta.TableMaxSeatCount,
		te.sm.CurrentDealerSeatID(),
		te.sm.CurrentSBSeatID(),
		te.sm.CurrentBBSeatID(),
		cloneTable.State.SeatMap,
		cloneTable.State.PlayerStates,
	)

	// update player positions
	te.updatePlayerPositions(cloneTable.Meta.TableMaxSeatCount, cloneTable.State.PlayerStates)

	// Step 6: 更新桌次狀態 (GameCount & 當前 Dealer & BB 位置)
	cloneTable.State.GameCount = cloneTable.State.GameCount + 1
	cloneTable.State.CurrentDealerSeat = te.sm.CurrentDealerSeatID()
	cloneTable.State.CurrentSBSeat = te.sm.CurrentSBSeatID()
	cloneTable.State.CurrentBBSeat = te.sm.CurrentBBSeatID()

	return cloneTable, nil
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
	if !funk.Contains(playerSettings[0].Positions, Position_Dealer) {
		playerSettings[0].Positions = append(playerSettings[0].Positions, Position_Dealer)
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
	te.game.OnAntesReceived(func(gs *pokerface.GameState) {
		for gpIdx, p := range gs.Players {
			if playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(gpIdx); playerIdx != UnsetValue {
				player := te.table.State.PlayerStates[playerIdx]
				pga := te.createPlayerGameAction(player.PlayerID, playerIdx, "pay", player.Bankroll, p)
				pga.Round = "ante"
				te.emitGamePlayerActionEvent(*pga)
			}
		}
	})
	te.game.OnBlindsReceived(func(gs *pokerface.GameState) {
		for gpIdx, p := range gs.Players {
			for _, pos := range p.Positions {
				if funk.Contains([]string{Position_SB, Position_BB}, pos) {
					if playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(gpIdx); playerIdx != UnsetValue {
						player := te.table.State.PlayerStates[playerIdx]
						pga := te.createPlayerGameAction(player.PlayerID, playerIdx, "pay", player.Bankroll, p)
						te.emitGamePlayerActionEvent(*pga)
					}
				}
			}
		}
	})
	te.game.OnGameRoundClosed(func(gs *pokerface.GameState) {
		te.table.State.CurrentActionEndAt = 0
	})

	// start game
	if _, err := te.game.Start(); err != nil {
		return err
	}

	te.table.State.Status = TableStateStatus_TableGamePlaying
	te.table.State.GameBlindState = &TableBlindState{
		Level:  blind.Level,
		Ante:   blind.Ante,
		Dealer: blind.Dealer,
		SB:     blind.SB,
		BB:     blind.BB,
	}
	return nil
}

func (te *tableEngine) settleGame() {
	te.table.State.Status = TableStateStatus_TableGameSettled

	// 計算攤牌勝率用
	notFoldCount := 0
	for _, result := range te.table.State.GameState.Result.Players {
		p := te.table.State.GameState.GetPlayer(result.Idx)
		if p != nil && !p.Fold {
			notFoldCount++
		}
	}

	// 計算贏家
	rank := settlement.NewRank()
	for _, player := range te.table.State.GameState.Players {
		if !player.Fold {
			rank.AddContributor(player.Combination.Power, player.Idx)
		}
	}
	rank.Calculate()
	winnerGamePlayerIndexes := rank.GetWinners()
	winnerPlayerIndexes := make(map[int]bool)
	for _, winnerGamePlayerIndex := range winnerGamePlayerIndexes {
		playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(winnerGamePlayerIndex)
		if playerIdx == UnsetValue {
			fmt.Printf("[DEBUGsettleGame] can't find player index from game player index (%d)", winnerGamePlayerIndex)
			continue
		}

		winnerPlayerIndexes[playerIdx] = true
	}

	// 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range te.table.State.GameState.Result.Players {
		playerIdx := te.table.State.GamePlayerIndexes[player.Idx]
		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.Bankroll = player.Final

		// 更新玩家攤牌勝率
		p := te.table.State.GameState.GetPlayer(player.Idx)
		if p != nil && !p.Fold && notFoldCount > 1 {
			playerState.GameStatistics.ShowdownWinningChance = true
			if _, exist := winnerPlayerIndexes[playerIdx]; exist {
				playerState.GameStatistics.IsShowdownWinning = true
			}
		} else {
			playerState.GameStatistics.ShowdownWinningChance = false
		}
	}

	// 更新 NextBBOrderPlayerIDs (移除沒有籌碼的玩家)
	te.table.State.NextBBOrderPlayerIDs = te.refreshNextBBOrderPlayerIDs(te.sm.CurrentBBSeatID(), te.table.Meta.TableMaxSeatCount, te.table.State.PlayerStates, te.table.State.SeatMap)

	te.emitEvent("SettleTableGameResult", "")
	te.emitTableStateEvent(TableStateEvent_GameSettled)
}

func (te *tableEngine) continueGame() error {
	// Reset table state
	te.table.State.Status = TableStateStatus_TableGameStandby
	te.table.State.GamePlayerIndexes = make([]int, 0)
	te.table.State.NextBBOrderPlayerIDs = make([]string, 0)
	te.table.State.CurrentActionEndAt = 0
	te.table.State.GameState = nil
	te.table.State.LastPlayerGameAction = nil
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		playerState := te.table.State.PlayerStates[i]
		playerState.Positions = make([]string, 0)
		playerState.GameStatistics = NewPlayerGameStatistics()
		if err := te.sm.UpdatePlayerHasChips(playerState.PlayerID, playerState.Bankroll > 0); err != nil {
			return err
		}
		active, err := te.sm.IsPlayerActive(playerState.PlayerID)
		if err != nil {
			return err
		}

		playerState.IsParticipated = active
	}

	var nextMoveInterval int
	var nextMoveHandler func() error

	// 桌次時間到了則不自動開下一手 (CT/Cash)
	ctMTTAutoGameOpenEnd := false
	if te.table.Meta.Mode == CompetitionMode_CT || te.table.Meta.Mode == CompetitionMode_Cash {
		tableEndAt := time.Unix(te.table.State.StartAt, 0).Add(time.Second * time.Duration(te.table.Meta.MaxDuration)).Unix()
		ctMTTAutoGameOpenEnd = time.Now().Unix() > tableEndAt
	}

	if ctMTTAutoGameOpenEnd {
		nextMoveInterval = 1
		nextMoveHandler = func() error {
			fmt.Printf("[DEBUG#continueGame] delay -> not auto opened %s table (%s), end: %s, now: %s\n", te.table.Meta.Mode, te.table.ID, time.Unix(te.table.State.StartAt, 0).Add(time.Second*time.Duration(te.table.Meta.MaxDuration)), time.Now())
			te.onAutoGameOpenEnd(te.table.Meta.CompetitionID, te.table.ID)
			return nil
		}
	} else {
		nextMoveInterval = te.options.Interval
		nextMoveHandler = func() error {
			// 如果在 Interval 這期間，該桌已關閉，則不繼續動作
			if te.table.State.Status == TableStateStatus_TableClosed {
				return nil
			}

			// 如果在 Interval 這期間，該桌已釋放，則不繼續動作
			if te.isReleased {
				return nil
			}

			// 桌次接續動作: pause or open
			if te.table.ShouldPause() {
				// 暫停處理
				te.table.State.Status = TableStateStatus_TablePausing
				te.emitEvent("ContinueGame -> Pause", "")
				te.emitTableStateEvent(TableStateEvent_StatusUpdated)
			} else {
				if te.shouldAutoGameOpen() {
					fmt.Println("[DEBUG#continueGame] delay -> TableGameOpen")
					return te.TableGameOpen()
				}

				// Unhandled Situation
				str, _ := te.table.GetJSON()
				fmt.Printf("[DEBUG#continueGame] delay -> unhandled issue. Table: %s\n", str)
			}
			return nil
		}

		te.rgForOpenGame.Stop()
		te.rgForOpenGame.OnCompleted(func(rg *syncsaga.ReadyGroup) {
			err := nextMoveHandler()
			if err != nil {
				fmt.Printf("[DEBUG#continueGame] rgForOpenGame.OnCompleted() -> nextMoveHandler error: %v\n", err)
			}
		})
		te.rgForOpenGame.ResetParticipants()
		for playerIdx := range te.table.State.PlayerStates {
			if te.table.State.PlayerStates[playerIdx].IsIn {
				// 目前入桌玩家才要放到 ready group 做處理
				te.rgForOpenGame.Add(int64(playerIdx), false)
			}
		}

		te.rgForOpenGame.Start()
	}

	return te.delay(nextMoveInterval, nextMoveHandler)
}

package pokertable

import (
	"fmt"
	"sync"
	"time"

	"github.com/thoas/go-funk"
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
		if event == pokerface.GameEvent_RoundClosed {
			te.table.State.LastPlayerGameAction = nil
		}
	}
}

func (te *tableEngine) openGame(oldTable *Table) (*Table, error) {
	// Step 1: Clone Table for calculation
	cloneTable, err := oldTable.Clone()
	if err != nil {
		return oldTable, err
	}

	// Step 2: 更新狀態
	cloneTable.State.Status = TableStateStatus_TableGameOpened

	// Step 3: 檢查參賽資格
	for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
		// 沒有入桌玩家直接不參加
		if !cloneTable.State.PlayerStates[i].IsIn {
			cloneTable.State.PlayerStates[i].IsParticipated = false
			continue
		}

		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if cloneTable.State.PlayerStates[i].IsParticipated || cloneTable.State.PlayerStates[i].IsBetweenDealerBB {
			cloneTable.State.PlayerStates[i].IsParticipated = cloneTable.State.PlayerStates[i].Bankroll > 0
			continue
		}

		// 檢查後手 (有錢的玩家可參賽)
		cloneTable.State.PlayerStates[i].IsParticipated = cloneTable.State.PlayerStates[i].Bankroll > 0
	}

	// Step 4: 處理可參賽玩家剩餘一人時，桌上有其他玩家情形
	if len(cloneTable.ParticipatedPlayers()) < cloneTable.Meta.TableMinPlayerCount {
		for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
			// 沒入桌或沒籌碼玩家不能玩
			if cloneTable.State.PlayerStates[i].Bankroll == 0 || !cloneTable.State.PlayerStates[i].IsIn {
				continue
			}

			cloneTable.State.PlayerStates[i].IsParticipated = true
			cloneTable.State.PlayerStates[i].IsBetweenDealerBB = false
		}
	}

	// Step 5: 計算新 Dealer Seat & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(cloneTable.State.GameCount, cloneTable.State.CurrentDealerSeat, cloneTable.Meta.TableMinPlayerCount, cloneTable.Meta.TableMaxSeatCount, cloneTable.State.PlayerStates, cloneTable.State.SeatMap)
	newDealerTableSeatIdx := cloneTable.State.PlayerStates[newDealerPlayerIdx].Seat

	// Step 6: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
		if !cloneTable.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if !cloneTable.State.PlayerStates[i].IsParticipated {
			continue
		}

		if newDealerTableSeatIdx-cloneTable.State.CurrentDealerSeat < 0 {
			for j := cloneTable.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx+cloneTable.Meta.TableMaxSeatCount; j++ {
				if (j % cloneTable.Meta.TableMaxSeatCount) != cloneTable.State.PlayerStates[i].Seat {
					continue
				}

				cloneTable.State.PlayerStates[i].IsBetweenDealerBB = false
				cloneTable.State.PlayerStates[i].IsParticipated = true
			}
		} else {
			for j := cloneTable.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx; j++ {
				if j != cloneTable.State.PlayerStates[i].Seat {
					continue
				}

				cloneTable.State.PlayerStates[i].IsBetweenDealerBB = false
				cloneTable.State.PlayerStates[i].IsParticipated = true
			}
		}
	}

	// Step 7: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	gamePlayerIndexes := FindGamePlayerIndexes(newDealerTableSeatIdx, cloneTable.State.SeatMap, cloneTable.State.PlayerStates)
	if len(gamePlayerIndexes) < cloneTable.Meta.TableMinPlayerCount {
		fmt.Printf("[DEBUG#MTT#openGame] Competition (%s), Table (%s), TableMinPlayerCount: %d, GamePlayerIndexes: %+v\n", cloneTable.Meta.CompetitionID, cloneTable.ID, cloneTable.Meta.TableMinPlayerCount, gamePlayerIndexes)
		json, _ := cloneTable.GetJSON()
		fmt.Println(json)
		return oldTable, ErrTableOpenGameFailed
	}
	cloneTable.State.GamePlayerIndexes = gamePlayerIndexes

	// Step 8: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(cloneTable.Meta.Rule, cloneTable.State.PlayerStates, cloneTable.State.GamePlayerIndexes)
	for playerIdx := 0; playerIdx < len(cloneTable.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && cloneTable.State.PlayerStates[playerIdx].IsParticipated {
			cloneTable.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 9: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置)
	cloneTable.State.GameCount = cloneTable.State.GameCount + 1
	cloneTable.State.CurrentDealerSeat = newDealerTableSeatIdx
	if len(gamePlayerIndexes) == 2 {
		bbPlayerIdx := gamePlayerIndexes[1]
		cloneTable.State.CurrentBBSeat = cloneTable.State.PlayerStates[bbPlayerIdx].Seat
	} else if len(gamePlayerIndexes) > 2 {
		bbPlayerIdx := gamePlayerIndexes[2]
		cloneTable.State.CurrentBBSeat = cloneTable.State.PlayerStates[bbPlayerIdx].Seat
	} else {
		cloneTable.State.CurrentBBSeat = UnsetValue
	}

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

		if te.table.State.PlayerStates[playerIdx].Bankroll == 0 {
			te.table.State.PlayerStates[playerIdx].IsParticipated = false
		}
	}

	// 更新 SeatChanges
	te.table.State.SeatChanges = te.calcSeatChanges(te.table)

	te.emitEvent("SettleTableGameResult", "")
	te.emitTableStateEvent(TableStateEvent_GameSettled)
}

func (te *tableEngine) continueGame() error {
	return te.delay(te.options.Interval, func() error {
		// Reset table state
		te.table.State.GamePlayerIndexes = []int{}
		te.table.State.GameState = nil
		te.table.State.SeatChanges = nil
		te.table.State.LastPlayerGameAction = nil
		for i := 0; i < len(te.table.State.PlayerStates); i++ {
			te.table.State.PlayerStates[i].Positions = make([]string, 0)
			te.table.State.PlayerStates[i].GameStatistics.ActionTimes = 0
			te.table.State.PlayerStates[i].GameStatistics.RaiseTimes = 0
			te.table.State.PlayerStates[i].GameStatistics.CallTimes = 0
			te.table.State.PlayerStates[i].GameStatistics.CheckTimes = 0
			te.table.State.PlayerStates[i].GameStatistics.IsFold = false
			te.table.State.PlayerStates[i].GameStatistics.FoldRound = ""
		}

		// 檢查是否暫停
		if te.table.ShouldPause() {
			// 暫停處理
			te.table.State.Status = TableStateStatus_TablePausing
			te.emitEvent("ContinueGame -> Pause", "")
			te.emitTableStateEvent(TableStateEvent_StatusUpdated)
		} else {
			// 自動開下一手條件: status = TableStateStatus_TableGameSettled 且有籌碼玩家 >= 最小開打人數
			if te.table.State.Status == TableStateStatus_TableGameSettled && len(te.table.AlivePlayers()) >= te.table.Meta.TableMinPlayerCount {
				return te.TableGameOpen()
			}
		}
		return nil
	})
}

func (te *tableEngine) onGameClosed() error {
	te.settleGame()
	return te.continueGame()
}

func (te *tableEngine) calcSeatChanges(oldTable *Table) *TableGameSeatChanges {
	te.lock.Lock()
	defer te.lock.Unlock()

	cloneTable, err := oldTable.Clone()
	if err != nil {
		te.emitErrorEvent("calculate seat changes clone old table", "", err)
		return nil
	}

	// Preparing seat changes
	sc := &TableGameSeatChanges{}

	// find no chips players
	leavePlayerIDs := make([]string, 0)
	alivePlayerIndexes := make([]int, 0)
	for playerIdx, player := range oldTable.State.PlayerStates {
		if player.Bankroll <= 0 {
			leavePlayerIDs = append(leavePlayerIDs, player.PlayerID)
		} else {
			alivePlayerIndexes = append(alivePlayerIndexes, playerIdx)
		}
	}

	// delete no chips players
	newPlayerStates, newSeatMap, newGamePlayerIndexes := te.calcLeavePlayers(cloneTable.State.Status, leavePlayerIDs, cloneTable.State.PlayerStates, cloneTable.Meta.TableMaxSeatCount)
	cloneTable.State.PlayerStates = newPlayerStates
	cloneTable.State.SeatMap = newSeatMap
	cloneTable.State.GamePlayerIndexes = newGamePlayerIndexes

	if len(cloneTable.State.PlayerStates) < 2 {
		sc.NewDealer = cloneTable.State.PlayerStates[0].Seat
		sc.NewSB = UnsetValue
		sc.NewBB = UnsetValue
	} else if len(alivePlayerIndexes) == 1 {
		sc.NewDealer = cloneTable.State.PlayerStates[alivePlayerIndexes[0]].Seat
		sc.NewSB = UnsetValue
		sc.NewBB = UnsetValue
	} else {
		// try open next game
		newTable, err := te.openGame(cloneTable)
		if err != nil {
			te.emitErrorEvent("calculate seat changes when try open game", "", err)
			return nil
		}

		// Update dealer, sb and bb
		newSBSeat := UnsetValue
		for _, player := range newTable.State.PlayerStates {
			if funk.Contains(player.Positions, Position_SB) {
				newSBSeat = player.Seat
				break
			}
		}

		sc.NewDealer = newTable.State.CurrentDealerSeat
		sc.NewSB = newSBSeat
		sc.NewBB = newTable.State.CurrentBBSeat
	}

	return sc
}

func (te *tableEngine) calcLeavePlayers(status TableStateStatus, leavePlayerIDs []string, currentPlayers []*TablePlayerState, tableMaxSeatCount int) ([]*TablePlayerState, []int, []int) {
	// calc delete target players in PlayerStates
	newPlayerStates := make([]*TablePlayerState, 0)
	for _, player := range currentPlayers {
		exist := funk.Contains(leavePlayerIDs, func(leavePlayerID string) bool {
			return player.PlayerID == leavePlayerID
		})
		if !exist {
			newPlayerStates = append(newPlayerStates, player)
		}
	}

	// calc seatMap
	newSeatMap := NewDefaultSeatMap(tableMaxSeatCount)
	for newPlayerIdx, player := range newPlayerStates {
		newSeatMap[player.Seat] = newPlayerIdx
	}

	// calc new gamePlayerIndexes
	newPlayerData := make(map[string]int)
	for newPlayerIdx, player := range newPlayerStates {
		newPlayerData[player.PlayerID] = newPlayerIdx
	}

	currentGamePlayerData := make(map[int]string) // key: currentPlayerIdx, value: currentPlayerID
	for _, playerIdx := range te.table.State.GamePlayerIndexes {
		currentGamePlayerData[playerIdx] = te.table.State.PlayerStates[playerIdx].PlayerID
	}
	gameStatuses := []TableStateStatus{
		TableStateStatus_TableGameOpened,
		TableStateStatus_TableGamePlaying,
		TableStateStatus_TableGameSettled,
	}
	newGamePlayerIndexes := make([]int, 0)
	if funk.Contains(gameStatuses, status) {
		for _, currentPlayerIdx := range te.table.State.GamePlayerIndexes {
			playerID := currentGamePlayerData[currentPlayerIdx]
			// sync newPlayerData player idx to newGamePlayerIndexes
			if newPlayerIdx, exist := newPlayerData[playerID]; exist {
				newGamePlayerIndexes = append(newGamePlayerIndexes, newPlayerIdx)
			}
		}
	} else {
		newGamePlayerIndexes = te.table.State.GamePlayerIndexes
	}

	return newPlayerStates, newSeatMap, newGamePlayerIndexes
}

func (te *tableEngine) createPlayerGameAction(playerID string, playerIdx int, action string, chips int64) *TablePlayerGameAction {
	pga := &TablePlayerGameAction{
		TableID:   te.table.ID,
		GameCount: te.table.State.GameCount,
		UpdateAt:  time.Now().Unix(),
		PlayerID:  playerID,
		Action:    action,
		Chips:     chips,
	}

	if te.table.State.GameState != nil {
		pga.GameID = te.table.State.GameState.GameID
		pga.Round = te.table.State.GameState.Status.Round
	}

	if playerIdx < len(te.table.State.PlayerStates) {
		pga.Seat = te.table.State.PlayerStates[playerIdx].Seat
		pga.Positions = te.table.State.PlayerStates[playerIdx].Positions
	}

	return pga
}

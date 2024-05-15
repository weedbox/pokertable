package pokertable

import (
	"fmt"
	"sync"
	"time"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokerface/settlement"
	"github.com/weedbox/syncsaga"
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

	if te.table.State.Status == TableStateStatus_TableGamePlaying {
		te.updateCurrentPlayerGameStatistics(gs)
	}

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
	// Step 1: Check TableState
	if !oldTable.State.BlindState.IsSet() {
		return oldTable, ErrTableOpenGameFailed
	}

	// Step 2: Clone Table for calculation
	cloneTable, err := oldTable.Clone()
	if err != nil {
		return oldTable, err
	}

	// Step 3: 更新狀態
	cloneTable.State.Status = TableStateStatus_TableGameOpened

	// Step 4: 檢查參賽資格
	for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
		playerState := cloneTable.State.PlayerStates[i]

		// 沒有入桌玩家直接不參加
		if !playerState.IsIn {
			playerState.IsParticipated = false
			continue
		}

		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if playerState.IsParticipated || playerState.IsBetweenDealerBB {
			playerState.IsParticipated = playerState.Bankroll > 0
			continue
		}

		// 檢查後手 (有錢的玩家可參賽)
		playerState.IsParticipated = playerState.Bankroll > 0
	}

	// Step 5: 處理可參賽玩家剩餘一人時，桌上有其他玩家情形
	if len(cloneTable.ParticipatedPlayers()) < cloneTable.Meta.TableMinPlayerCount {
		for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
			playerState := cloneTable.State.PlayerStates[i]

			// 沒入桌或沒籌碼玩家不能玩
			if playerState.Bankroll == 0 || !playerState.IsIn {
				continue
			}

			playerState.IsParticipated = true
			playerState.IsBetweenDealerBB = false
		}
	}

	// Step 6: 計算新 Dealer Seat & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(cloneTable.State.GameCount, cloneTable.State.CurrentDealerSeat, cloneTable.Meta.TableMinPlayerCount, cloneTable.Meta.TableMaxSeatCount, cloneTable.State.PlayerStates, cloneTable.State.SeatMap)
	newDealerTableSeatIdx := cloneTable.State.PlayerStates[newDealerPlayerIdx].Seat

	// Step 7: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(cloneTable.State.PlayerStates); i++ {
		playerState := cloneTable.State.PlayerStates[i]

		if !playerState.IsBetweenDealerBB {
			continue
		}

		if !playerState.IsParticipated {
			continue
		}

		if newDealerTableSeatIdx-cloneTable.State.CurrentDealerSeat < 0 {
			for j := cloneTable.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx+cloneTable.Meta.TableMaxSeatCount; j++ {
				if (j % cloneTable.Meta.TableMaxSeatCount) != playerState.Seat {
					continue
				}

				playerState.IsBetweenDealerBB = false
				playerState.IsParticipated = true
			}
		} else {
			for j := cloneTable.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx; j++ {
				if j != playerState.Seat {
					continue
				}

				playerState.IsBetweenDealerBB = false
				playerState.IsParticipated = true
			}
		}
	}

	// Step 8: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	gamePlayerIndexes := FindGamePlayerIndexes(newDealerTableSeatIdx, cloneTable.State.SeatMap, cloneTable.State.PlayerStates)
	if len(gamePlayerIndexes) < cloneTable.Meta.TableMinPlayerCount {
		fmt.Printf("[DEBUG#MTT#openGame] Competition (%s), Table (%s), TableMinPlayerCount: %d, GamePlayerIndexes: %+v\n", cloneTable.Meta.CompetitionID, cloneTable.ID, cloneTable.Meta.TableMinPlayerCount, gamePlayerIndexes)
		json, _ := cloneTable.GetJSON()
		fmt.Println(json)
		return oldTable, ErrTableOpenGameFailed
	}
	cloneTable.State.GamePlayerIndexes = gamePlayerIndexes

	// Step 9: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(cloneTable.Meta.Rule, cloneTable.State.PlayerStates, cloneTable.State.GamePlayerIndexes)
	for playerIdx := 0; playerIdx < len(cloneTable.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && cloneTable.State.PlayerStates[playerIdx].IsParticipated {
			cloneTable.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 10: 更新桌次狀態 (GameCount & 當前 Dealer & BB 位置)
	cloneTable.State.GameCount = cloneTable.State.GameCount + 1
	cloneTable.State.CurrentDealerSeat = newDealerTableSeatIdx
	if len(gamePlayerIndexes) == 2 {
		bbPlayer := cloneTable.State.PlayerStates[gamePlayerIndexes[1]]
		cloneTable.State.CurrentBBSeat = bbPlayer.Seat
	} else if len(gamePlayerIndexes) > 2 {
		gameBBPlayerIdx := 2
		bbPlayer := cloneTable.State.PlayerStates[gamePlayerIndexes[gameBBPlayerIdx]]
		cloneTable.State.CurrentBBSeat = bbPlayer.Seat
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
	if _, err := te.game.Start(); err != nil {
		return err
	}

	te.table.State.Status = TableStateStatus_TableGamePlaying
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
			fmt.Printf("[DEBUG#settleGame] can't find player index from game player index (%d)", winnerGamePlayerIndex)
			continue
		}

		winnerPlayerIndexes[playerIdx] = true
	}

	// 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range te.table.State.GameState.Result.Players {
		playerIdx := te.table.State.GamePlayerIndexes[player.Idx]
		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.Bankroll = player.Final
		if playerState.Bankroll == 0 {
			playerState.IsParticipated = false
		}

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
	te.table.State.NextBBOrderPlayerIDs = te.refreshNextBBOrderPlayerIDs(te.table.State.PlayerStates, te.table.State.GamePlayerIndexes)

	te.emitEvent("SettleTableGameResult", "")
	te.emitTableStateEvent(TableStateEvent_GameSettled)
}

func (te *tableEngine) continueGame() error {
	// Reset table state
	te.table.State.Status = TableStateStatus_TableGameStandby
	te.table.State.GamePlayerIndexes = make([]int, 0)
	te.table.State.NextBBOrderPlayerIDs = make([]string, 0)
	te.table.State.GameState = nil
	te.table.State.LastPlayerGameAction = nil
	for i := 0; i < len(te.table.State.PlayerStates); i++ {
		playerState := te.table.State.PlayerStates[i]
		playerState.Positions = make([]string, 0)
		playerState.GameStatistics = NewPlayerGameStatistics()
	}

	return te.delay(te.options.Interval, func() error {
		// 如果在 Interval 這期間，該桌已關閉，則不繼續動作
		if te.table.State.Status == TableStateStatus_TableClosed {
			return nil
		}

		// 如果在 Interval 這期間，該桌已釋放，則不繼續動作
		if te.isReleased {
			return nil
		}

		// 桌次時間到了則不自動開下一手 (CT/Cash)
		autoGameOpenEnd := false
		if te.table.Meta.Mode == "ct" || te.table.Meta.Mode == "cash" {
			tableEndAt := time.Unix(te.table.State.StartAt, 0).Add(time.Second * time.Duration(te.table.Meta.MaxDuration)).Unix()
			autoGameOpenEnd = time.Now().Unix() > tableEndAt
		}

		if autoGameOpenEnd {
			fmt.Printf("[DEBUG#continueGame] delay -> not auto opened %s table (%s), end: %s, now: %s\n", te.table.Meta.Mode, te.table.ID, time.Unix(te.table.State.StartAt, 0).Add(time.Second*time.Duration(te.table.Meta.MaxDuration)), time.Now())
			te.onAutoGameOpenEnd(te.table.Meta.CompetitionID, te.table.ID)
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
	})
}

func (te *tableEngine) shouldAutoGameOpen() bool {
	// 自動開下一手條件: status = TableStateStatus_TableGameStandby 且有籌碼玩家 >= 最小開打人數
	return te.table.State.Status == TableStateStatus_TableGameStandby &&
		len(te.table.AlivePlayers()) >= te.table.Meta.TableMinPlayerCount
}

func (te *tableEngine) onGameClosed() error {
	te.settleGame()
	return te.continueGame()
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

func (te *tableEngine) createPlayerGameAction(playerID string, playerIdx int, action string, chips int64, player *pokerface.PlayerState) *TablePlayerGameAction {
	pga := &TablePlayerGameAction{
		CompetitionID: te.table.Meta.CompetitionID,
		TableID:       te.table.ID,
		GameCount:     te.table.State.GameCount,
		UpdateAt:      time.Now().Unix(),
		PlayerID:      playerID,
		Action:        action,
		Chips:         chips,
	}

	if te.table.State.GameState != nil {
		pga.GameID = te.table.State.GameState.GameID
		pga.Round = te.table.State.GameState.Status.Round
	}

	if playerIdx < len(te.table.State.PlayerStates) {
		pga.Seat = te.table.State.PlayerStates[playerIdx].Seat
		pga.Positions = te.table.State.PlayerStates[playerIdx].Positions
	}

	if player != nil {
		pga.Bankroll = player.Bankroll
		pga.InitialStackSize = player.InitialStackSize
		pga.StackSize = player.StackSize
		pga.Pot = player.Pot
		pga.Wager = player.Wager
	}

	return pga
}

func (te *tableEngine) batchAddPlayers(players []JoinPlayer) error {
	// decide seats
	availableSeats, err := RandomSeats(te.table.State.SeatMap, len(players))
	if err != nil {
		return err
	}

	// update table state
	newSeatMap := make([]int, len(te.table.State.SeatMap))
	copy(newSeatMap, te.table.State.SeatMap)
	newPlayers := make([]*TablePlayerState, 0)
	for idx, player := range players {
		reservedSeat := player.Seat

		// add new player
		var seat int
		if reservedSeat == UnsetValue {
			seat = availableSeats[idx]
		} else {
			if te.table.State.SeatMap[reservedSeat] == UnsetValue {
				seat = reservedSeat
			} else {
				return ErrTablePlayerSeatUnavailable
			}
		}

		// update state
		player := &TablePlayerState{
			PlayerID:          player.PlayerID,
			Seat:              seat,
			Positions:         []string{Position_Unknown},
			IsParticipated:    false,
			IsBetweenDealerBB: IsBetweenDealerBB(seat, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.TableMaxSeatCount, te.table.Meta.Rule),
			Bankroll:          player.RedeemChips,
			IsIn:              false,
			GameStatistics:    NewPlayerGameStatistics(),
		}
		newPlayers = append(newPlayers, player)

		newPlayerIdx := len(te.table.State.PlayerStates) + len(newPlayers) - 1
		newSeatMap[seat] = newPlayerIdx
	}

	te.table.State.SeatMap = newSeatMap
	te.table.State.PlayerStates = append(te.table.State.PlayerStates, newPlayers...)

	// 如果時間到了還沒有入座則自動入座
	te.playersAutoIn()

	// emit events
	for _, player := range newPlayers {
		te.emitTablePlayerStateEvent(player)
		te.emitTablePlayerReservedEvent(player)
	}

	return nil
}

func (te *tableEngine) playersAutoIn() {
	// Preparing ready group for waiting all players' join
	te.rg.Stop()
	te.rg.SetTimeoutInterval(15)
	te.rg.OnTimeout(func(rg *syncsaga.ReadyGroup) {
		// Auto Ready By Default
		states := rg.GetParticipantStates()
		for playerIdx, isReady := range states {
			if !isReady {
				rg.Ready(playerIdx)
			}
		}
	})
	te.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		isInCount := 0
		alivePlayers := 0
		for playerIdx, player := range te.table.State.PlayerStates {
			// 如果時間到了還沒有入座則自動入座
			if !player.IsIn {
				te.table.State.PlayerStates[playerIdx].IsIn = true
			}

			if te.table.State.PlayerStates[playerIdx].IsIn {
				isInCount++
			}

			if te.table.State.PlayerStates[playerIdx].Bankroll > 0 {
				alivePlayers++
			}
		}

		// 等所有玩家 is_in 且大於開打人數，且未開始 game，則開始遊戲
		gameStartingStatuses := []TableStateStatus{
			TableStateStatus_TableGameOpened,
			TableStateStatus_TableGamePlaying,
			TableStateStatus_TableGameSettled,
			TableStateStatus_TableGameStandby,
		}
		isGameRunning := funk.Contains(gameStartingStatuses, te.table.State.Status)
		if isInCount >= 2 && !isGameRunning {
			if te.table.State.GameCount == 0 {
				if err := te.StartTableGame(); err != nil {
					te.emitErrorEvent("StartTableGame", "", err)
				}
			} else if te.table.State.GameCount > 0 && te.table.State.BlindState.Level > 0 && alivePlayers >= 2 {
				// 中場休息時，不會開下一手，等到中場休息結束後，外部才會呼叫開下一手
				fmt.Println("[DEBUG#playersAutoIn] OnCompleted -> TableGameOpen")
				if err := te.TableGameOpen(); err != nil {
					te.emitErrorEvent("TableGameOpen", "", err)
				}
			}
		}
	})

	te.rg.ResetParticipants()
	for playerIdx := range te.table.State.PlayerStates {
		if !te.table.State.PlayerStates[playerIdx].IsIn {
			// 新加入的玩家才要放到 ready group 做處理
			te.rg.Add(int64(playerIdx), false)
		}
	}

	te.rg.Start()
}

func (te *tableEngine) batchRemovePlayers(playerIDs []string) {
	newPlayerStates, newSeatMap, newGamePlayerIndexes := te.calcLeavePlayers(te.table.State.Status, playerIDs, te.table.State.PlayerStates, te.table.Meta.TableMaxSeatCount)
	te.table.State.PlayerStates = newPlayerStates
	te.table.State.SeatMap = newSeatMap
	te.table.State.GamePlayerIndexes = newGamePlayerIndexes
}

func (te *tableEngine) refreshNextBBOrderPlayerIDs(players []*TablePlayerState, gamePlayerIndexes []int) []string {
	nextBBOrderPlayerIDs := make([]string, 0)

	targetNextBBOrderPlayerIDs := make(map[string]bool, 0)
	for _, p := range players {
		// 只挑有籌碼的玩家進入 NextBBOrderPlayerIDs 清單
		if p.Bankroll > 0 {
			targetNextBBOrderPlayerIDs[p.PlayerID] = false
		}
	}

	if len(gamePlayerIndexes) == 2 {
		if dealerPlayer := players[gamePlayerIndexes[0]]; dealerPlayer.Bankroll > 0 {
			nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, dealerPlayer.PlayerID)
			targetNextBBOrderPlayerIDs[dealerPlayer.PlayerID] = true
		}

		if bbPlayer := players[gamePlayerIndexes[1]]; bbPlayer.Bankroll > 0 {
			nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, bbPlayer.PlayerID)
			targetNextBBOrderPlayerIDs[bbPlayer.PlayerID] = true
		}
	} else if len(gamePlayerIndexes) > 2 {
		gameBBPlayerIdx := 2

		// starts with UG GamePlayerIndex (ug next round will be bb)
		for i := gameBBPlayerIdx + 1; i < len(gamePlayerIndexes)+gameBBPlayerIdx+1; i++ {
			gpIdx := i % len(gamePlayerIndexes)
			player := players[gamePlayerIndexes[gpIdx]]

			// 只挑有籌碼的玩家進入 NextBBOrderPlayerIDs 清單
			if player.Bankroll > 0 {
				nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, player.PlayerID)
				targetNextBBOrderPlayerIDs[player.PlayerID] = true
			}
		}
	}

	// 如果有籌碼但沒有參與的玩家，加入 NextBBOrderPlayerIDs 的後面位置
	for playerID, picked := range targetNextBBOrderPlayerIDs {
		if !picked {
			nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, playerID)
		}
	}

	return nextBBOrderPlayerIDs
}

package pokertable

import (
	"fmt"
	"sync"
	"time"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
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

	// Step 10: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置, 下一手 BB 座位玩家索引值陣列)
	cloneTable.State.GameCount = cloneTable.State.GameCount + 1
	cloneTable.State.CurrentDealerSeat = newDealerTableSeatIdx
	if len(gamePlayerIndexes) == 2 {
		dealerPlayer := cloneTable.State.PlayerStates[gamePlayerIndexes[0]]
		bbPlayer := cloneTable.State.PlayerStates[gamePlayerIndexes[1]]
		cloneTable.State.CurrentBBSeat = bbPlayer.Seat
		cloneTable.State.NextBBOrderPlayerIDs = []string{dealerPlayer.PlayerID, bbPlayer.PlayerID}
	} else if len(gamePlayerIndexes) > 2 {
		gameBBPlayerIdx := 2
		bbPlayer := cloneTable.State.PlayerStates[gamePlayerIndexes[gameBBPlayerIdx]]
		cloneTable.State.CurrentBBSeat = bbPlayer.Seat

		targetNextBBOrderPlayerIDs := make(map[string]bool, 0)
		for _, p := range cloneTable.State.PlayerStates {
			targetNextBBOrderPlayerIDs[p.PlayerID] = false
		}

		// starts with UG GamePlayerIndex (ug next round will be bb)
		nextBBOrderPlayerIDs := make([]string, 0)
		for i := gameBBPlayerIdx + 1; i < len(gamePlayerIndexes)+gameBBPlayerIdx+1; i++ {
			gpIdx := i % len(gamePlayerIndexes)
			player := cloneTable.State.PlayerStates[gamePlayerIndexes[gpIdx]]
			nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, player.PlayerID)
		}

		// 如果有籌碼但沒有參與的玩家，加入 NextBBOrderPlayerIDs 的後面位置
		for playerID, picked := range targetNextBBOrderPlayerIDs {
			if !picked {
				nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, playerID)
			}
		}

		cloneTable.State.NextBBOrderPlayerIDs = nextBBOrderPlayerIDs
	} else {
		cloneTable.State.CurrentBBSeat = UnsetValue
		cloneTable.State.NextBBOrderPlayerIDs = []string{}
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

	// 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range te.table.State.GameState.Result.Players {
		playerIdx := te.table.State.GamePlayerIndexes[player.Idx]
		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.Bankroll = player.Final
		if playerState.Bankroll == 0 {
			playerState.IsParticipated = false
		}
	}

	// 更新 NextBBOrderPlayerIDs (移除沒有籌碼的玩家)
	newNextBBOrderPlayerIDs := make([]string, 0)
	for _, playerID := range te.table.State.NextBBOrderPlayerIDs {
		if playerIdx := te.table.FindPlayerIdx(playerID); playerIdx != -1 {
			if te.table.State.PlayerStates[playerIdx].Bankroll > 0 {
				newNextBBOrderPlayerIDs = append(newNextBBOrderPlayerIDs, playerID)
			}
		}
	}
	te.table.State.NextBBOrderPlayerIDs = newNextBBOrderPlayerIDs

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
		playerState.GameStatistics.ActionTimes = 0
		playerState.GameStatistics.RaiseTimes = 0
		playerState.GameStatistics.CallTimes = 0
		playerState.GameStatistics.CheckTimes = 0
		playerState.GameStatistics.IsFold = false
		playerState.GameStatistics.FoldRound = ""
	}

	return te.delay(te.options.Interval, func() error {
		// 如果在 Interval 這期間，該桌已關閉，則不繼續動作
		if te.table.State.Status == TableStateStatus_TableClosed {
			return nil
		}

		// 桌次接續動作: pause or open
		if te.table.ShouldPause() {
			// 暫停處理
			te.table.State.Status = TableStateStatus_TablePausing
			te.emitEvent("ContinueGame -> Pause", "")
			te.emitTableStateEvent(TableStateEvent_StatusUpdated)
		} else {
			// 自動開下一手條件: status = TableStateStatus_TableGameStandby 且有籌碼玩家 >= 最小開打人數
			if te.table.State.Status == TableStateStatus_TableGameStandby && len(te.table.AlivePlayers()) >= te.table.Meta.TableMinPlayerCount {
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
			GameStatistics:    TablePlayerGameStatistics{},
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
		for playerIdx, player := range te.table.State.PlayerStates {
			// 如果時間到了還沒有入座則自動入座
			if !player.IsIn {
				te.table.State.PlayerStates[playerIdx].IsIn = true
			}
		}

		if te.table.State.GameCount <= 0 {
			// 拆併桌起新桌，時間到了自動開打
			if err := te.StartTableGame(); err != nil {
				te.emitErrorEvent("StartTableGame", "", err)
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

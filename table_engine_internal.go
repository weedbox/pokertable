package pokertable

import (
	"fmt"
	"sync"
	"time"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokertable/seat_manager"
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

	te.tbForOpenGame.NewTask(time.Duration(interval)*time.Second, func(isCancelled bool) {
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
		te.updateCurrentActionEndAt(event, gs)
		te.emitEvent(gs.Status.CurrentEvent, "")
		te.emitTableStateEvent(TableStateEvent_GameUpdated)
		if event == pokerface.GameEvent_RoundClosed {
			te.table.State.LastPlayerGameAction = nil
		}
	}
}

func (te *tableEngine) updateCurrentActionEndAt(event pokerface.GameEvent, gs *pokerface.GameState) {
	p := gs.GetPlayer(gs.Status.CurrentPlayer)
	validRounds := []string{GameRound_Preflop, GameRound_Flop, GameRound_Turn, GameRound_River}
	validRoundState := te.table.State.Status == TableStateStatus_TableGamePlaying && event == pokerface.GameEvent_RoundStarted && funk.Contains(validRounds, gs.Status.Round)
	validActions := []string{WagerAction_Call, WagerAction_Raise, WagerAction_AllIn, WagerAction_Check, WagerAction_Fold, WagerAction_Bet}

	isActionValid := true
	for _, action := range p.AllowedActions {
		if !funk.Contains(validActions, action) {
			isActionValid = false
			break
		}
	}

	playerUnmoved := len(p.AllowedActions) > 0 && !p.Acted
	if validRoundState && playerUnmoved && isActionValid {
		te.table.State.CurrentActionEndAt = time.Now().Add(time.Second * time.Duration(te.table.Meta.ActionTime)).Unix()
	}
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
	playerSeatIDs := make(map[string]int)
	playerRandomSeatIDs := make([]string, 0)

	for _, p := range players {
		if p.Seat == seat_manager.UnsetSeatID {
			playerRandomSeatIDs = append(playerRandomSeatIDs, p.PlayerID)
		} else {
			playerSeatIDs[p.PlayerID] = p.Seat
		}
	}

	// update to seat manager
	if len(playerSeatIDs) > 0 {
		if err := te.sm.AssignSeats(playerSeatIDs); err != nil {
			return err
		}
	}

	if len(playerRandomSeatIDs) > 0 {
		if err := te.sm.RandomAssignSeats(playerRandomSeatIDs); err != nil {
			return err
		}
	}

	// update table state
	newSeatMap := make([]int, len(te.table.State.SeatMap))
	copy(newSeatMap, te.table.State.SeatMap)
	newPlayers := make([]*TablePlayerState, 0)
	for _, player := range players {
		// add new player
		seat, err := te.sm.GetSeatID(player.PlayerID)
		if err != nil {
			return err
		}

		// update state
		player := &TablePlayerState{
			PlayerID:       player.PlayerID,
			Seat:           seat,
			Positions:      []string{},
			IsParticipated: false,
			Bankroll:       player.RedeemChips,
			IsIn:           false,
			GameStatistics: NewPlayerGameStatistics(),
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
	te.rg.SetTimeoutInterval(17)
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
				te.PlayerJoin(player.PlayerID)
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
		// 非中場休息，有活著的玩家，且未開始遊戲
		if isInCount >= 2 && alivePlayers >= 2 && !isGameRunning && te.table.State.BlindState.Level > 0 {
			if te.table.State.GameCount == 0 {
				// 尚未開第一手，StartTableGame (MTT Only, CT 是由 competition 決定開始)
				if te.table.Meta.Mode == CompetitionMode_MTT {
					if err := te.StartTableGame(); err != nil {
						te.emitErrorEvent("StartTableGame", "", err)
					}
				}
			} else if te.table.State.GameCount > 0 {
				// 回復遊戲，TableGameOpen
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

func (te *tableEngine) batchRemovePlayers(playerIDs []string) error {
	newPlayerStates, newSeatMap, newGamePlayerIndexes := te.calcLeavePlayers(te.table.State.Status, playerIDs, te.table.State.PlayerStates, te.table.Meta.TableMaxSeatCount)
	te.table.State.PlayerStates = newPlayerStates
	te.table.State.SeatMap = newSeatMap
	te.table.State.GamePlayerIndexes = newGamePlayerIndexes
	return te.sm.RemoveSeats(playerIDs)
}

func (te *tableEngine) refreshNextBBOrderPlayerIDs(currentBBSeatID, tableMaxSeatCount int, players []*TablePlayerState, seatMap []int) []string {
	nextBBOrderPlayerIDs := make([]string, 0)
	for i := currentBBSeatID + 1; i <= tableMaxSeatCount+currentBBSeatID; i++ {
		newBBSeatID := i % tableMaxSeatCount
		playerIdx := seatMap[newBBSeatID]
		if playerIdx >= 0 && players[playerIdx].Bankroll > 0 {
			nextBBOrderPlayerIDs = append(nextBBOrderPlayerIDs, players[playerIdx].PlayerID)
		}
	}
	return nextBBOrderPlayerIDs
}

func (te *tableEngine) calcGamePlayerIndexes(rule string, maxSeatCount, currentDealerSeatID, currentSBSeatID, currentBBSeatID int, seatMap []int, players []*TablePlayerState) []int {
	playerLen := len(players)
	gamePlayerIndexes := make([]int, 0)
	playerPositions := make(map[int][]string) // key: player_index, value: positions
	if rule == CompetitionRule_ShortDeck {
		dealerPlayerIdx := seatMap[currentDealerSeatID]
		for i := dealerPlayerIdx; i < playerLen+dealerPlayerIdx; i++ {
			playerIdx := i % playerLen
			if players[playerIdx].IsParticipated {
				positions := make([]string, 0)
				if i == dealerPlayerIdx {
					positions = append(positions, Position_Dealer)
				}
				playerPositions[playerIdx] = positions
				gamePlayerIndexes = append(gamePlayerIndexes, playerIdx)
			}
		}
	} else {
		// CompetitionRule_Default & CompetitionRule_Omaha
		dealerPlayerIdx := UnsetValue // allow empty
		sbPlayerIdx := UnsetValue     // allow empty
		for idx, p := range players {
			if !p.IsParticipated {
				continue
			}

			if p.Seat == currentDealerSeatID {
				dealerPlayerIdx = idx
			}

			if p.Seat == currentSBSeatID {
				sbPlayerIdx = idx
			}
		}

		// find by dealer or empty-dealer situation
		if dealerPlayerIdx == UnsetValue {
			// find fake dealer index
			fakeDealerSeatID := UnsetValue
			startSeatID := UnsetValue
			if sbPlayerIdx == UnsetValue {
				// empty sb, use prior bb active player as fake dealer
				startSeatID = currentBBSeatID
			} else {
				// has sb, use prior sb active player as fake dealer
				startSeatID = currentSBSeatID
			}

			// find fake dealer seat id
			for i := startSeatID + maxSeatCount - 1; i >= startSeatID; i-- {
				seatID := i % maxSeatCount
				if sp, ok := te.sm.Seats()[seatID]; ok && sp != nil && sp.Active() {
					fakeDealerSeatID = seatID
					break
				}
			}

			// create game player indexes (starts at fake dealer player index)
			for i := fakeDealerSeatID; i < len(seatMap)+fakeDealerSeatID; i++ {
				seatID := i % len(seatMap)
				playerIdx := seatMap[seatID]
				if playerIdx >= 0 && players[playerIdx].IsParticipated {
					gamePlayerIndexes = append(gamePlayerIndexes, playerIdx)
				}
			}
		} else {
			startSeatID := currentDealerSeatID
			for i := startSeatID; i < len(seatMap)+startSeatID; i++ {
				seatID := i % len(seatMap)
				playerIdx := seatMap[seatID]
				if playerIdx >= 0 && players[playerIdx].IsParticipated {
					gamePlayerIndexes = append(gamePlayerIndexes, playerIdx)
				}
			}
		}
	}

	return gamePlayerIndexes
}

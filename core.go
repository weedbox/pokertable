package pokertable

import (
	"strings"
	"time"

	"github.com/weedbox/pokerface"
	"github.com/weedbox/syncsaga"
	"github.com/weedbox/timebank"
)

func (te *tableEngine) incomingRequest(tableID string, action RequestAction, param interface{}) error {
	tableGame, exist := te.tableGames.Load(tableID)
	if !exist {
		return ErrTableNotFound
	}

	te.incoming <- &Request{
		Action: action,
		Payload: Payload{
			TableGame: tableGame.(*TableGame),
			Param:     param,
		},
	}

	return nil
}

func (te *tableEngine) emitEvent(eventName string, playerID string, table *Table) {
	table.RefreshUpdateAt()
	// fmt.Printf("->[#%d][%d][%s] emit Event: %s\n", table.UpdateSerial, table.State.GameCount, playerID, eventName)
	te.onTableUpdated(table)
}

func (te *tableEngine) emitErrorEvent(eventName RequestAction, playerID string, err error, table *Table) {
	table.RefreshUpdateAt()
	// fmt.Printf("->[#%d][%d][%s] emit ERROR Event: %s, Error: %v\n", table.UpdateSerial, table.State.GameCount, playerID, eventName, err)
	te.onErrorUpdated(err)
}

func (te *tableEngine) run() {
	for req := range te.incoming {
		te.requestHandler(req)
	}
}

func (te *tableEngine) requestHandler(req *Request) {
	handlers := map[RequestAction]func(Payload){
		RequestAction_BalanceTable:      te.handleBalanceTable,
		RequestAction_DeleteTable:       te.handleDeleteTable,
		RequestAction_StartTableGame:    te.handleStartTableGame,
		RequestAction_TableGameOpen:     te.handleTableGameOpen,
		RequestAction_PlayerJoin:        te.handlePlayerJoin,
		RequestAction_PlayerRedeemChips: te.handlePlayerRedeemChips,
		RequestAction_PlayersLeave:      te.handlePlayersLeave,
		RequestAction_PlayerReady:       te.handlePlayerReady,
		RequestAction_PlayerPay:         te.handlePlayerPay,
		RequestAction_PlayerBet:         te.handlePlayerBet,
		RequestAction_PlayerRaise:       te.handlePlayerRaise,
		RequestAction_PlayerCall:        te.handlePlayerCall,
		RequestAction_PlayerAllin:       te.handlePlayerAllin,
		RequestAction_PlayerCheck:       te.handlePlayerCheck,
		RequestAction_PlayerFold:        te.handlePlayerFold,
		RequestAction_PlayerPass:        te.handlePlayerPass,
	}

	handler, ok := handlers[req.Action]
	if !ok {
		return
	}
	handler(req.Payload)
}

func (te *tableEngine) handleBalanceTable(payload Payload) {
	payload.TableGame.Table.State.Status = TableStateStatus_TableBalancing

	te.emitEvent("BalanceTable", "", payload.TableGame.Table)
}

func (te *tableEngine) handleDeleteTable(payload Payload) {
	tableID := payload.TableGame.Table.ID

	payload.TableGame.Table.State.Status = TableStateStatus_TableClosed

	te.emitEvent("DeleteTable", "", payload.TableGame.Table)

	// update tableGames
	te.tableGames.Delete(tableID)
}

func (te *tableEngine) handleStartTableGame(payload Payload) {
	// 初始化桌 & 開局 & 開始遊戲
	payload.TableGame.Table.State.StartAt = time.Now().Unix()
	payload.TableGame.Table.ActivateBlindState()

	te.handleTableGameOpen(payload)
}

func (te *tableEngine) handleTableGameOpen(payload Payload) {
	// 開局
	payload.TableGame.Table.OpenGame()

	te.emitEvent("TableGameOpen", "", payload.TableGame.Table)

	// 開始遊戲
	if err := te.startGame(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_StartTableGame, "", err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) handlePlayerJoin(payload Payload) {
	joinPlayer := payload.Param.(JoinPlayer)

	if err := payload.TableGame.Table.PlayerJoin(joinPlayer.PlayerID, joinPlayer.RedeemChips); err != nil {
		te.emitErrorEvent(RequestAction_PlayerJoin, joinPlayer.PlayerID, err, payload.TableGame.Table)
		return
	}

	te.emitEvent("PlayerJoin", joinPlayer.PlayerID, payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerRedeemChips(payload Payload) {
	joinPlayer := payload.Param.(JoinPlayer)

	// find player index in PlayerStates
	playerIdx := payload.TableGame.Table.findPlayerIdx(joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		te.emitErrorEvent(RequestAction_PlayerRedeemChips, joinPlayer.PlayerID, ErrPlayerNotFound, payload.TableGame.Table)
		return
	}

	payload.TableGame.Table.PlayerRedeemChips(playerIdx, joinPlayer.RedeemChips)

	te.emitEvent("PlayerRedeemChips", joinPlayer.PlayerID, payload.TableGame.Table)
}

func (te *tableEngine) handlePlayersLeave(payload Payload) {
	playerIDs := payload.Param.([]string)

	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := te.findPlayerIdx(payload.TableGame.Table.State.PlayerStates, playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return
	}

	payload.TableGame.Table.PlayersLeave(leavePlayerIndexes)

	te.emitEvent("PlayersLeave", strings.Join(playerIDs, ","), payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerReady(payload Payload) {
	playerID := payload.Param.(string)

	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(payload.TableGame.Table.State.PlayerStates, payload.TableGame.Table.State.GamePlayerIndexes, playerID)
	if gamePlayerIdx == UnsetValue {
		te.emitErrorEvent(RequestAction_PlayerReady, playerID, ErrPlayerNotFound, payload.TableGame.Table)
		return
	}

	// handle ready group
	rg, exist := payload.TableGame.GamePlayerReadies[payload.TableGame.Game.GetState().GameID]
	if !exist {
		te.emitErrorEvent(RequestAction_PlayerReady, playerID, ErrInvalidReadyAction, payload.TableGame.Table)
		return
	}
	rg.Ready(int64(gamePlayerIdx))
}

func (te *tableEngine) handlePlayerPay(payload Payload) {
	param := payload.Param.(PlayerPayParam)

	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(payload.TableGame.Table.State.PlayerStates, payload.TableGame.Table.State.GamePlayerIndexes, param.PlayerID)
	if gamePlayerIdx == UnsetValue {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, ErrPlayerNotFound, payload.TableGame.Table)
		return
	}

	// handle ready group
	gs := payload.TableGame.Game.GetState()
	event := gs.Status.CurrentEvent.Name
	// Pay Ante: call pay ante ready group ready
	if param.Chips == gs.Meta.Ante && event == GameEventName(pokerface.GameEvent_Prepared) {
		rg, exist := payload.TableGame.GamePlayerPayAnte[payload.TableGame.Game.GetState().GameID]
		if !exist {
			te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, ErrInvalidPayAnteAction, payload.TableGame.Table)
			return
		}
		rg.Ready(int64(gamePlayerIdx))
		return
	}

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, param.PlayerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Pay(param.Chips); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	// After Pay BB: run readies ready group
	if param.Chips == gs.Meta.Blind.BB && event == GameEventName(pokerface.GameEvent_RoundInitialized) {
		te.runPlayerReadiesCheck(gs.GameID, payload.TableGame)
	}

	te.emitEvent("PlayerPay", param.PlayerID, payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerBet(payload Payload) {
	param := payload.Param.(PlayerBetParam)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, param.PlayerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerBet, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Bet(param.Chips); err != nil {
		te.emitErrorEvent(RequestAction_PlayerBet, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	te.emitEvent("PlayerBet", param.PlayerID, payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerRaise(payload Payload) {
	param := payload.Param.(PlayerRaiseParam)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, param.PlayerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerRaise, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Raise(param.ChipLevel); err != nil {
		te.emitErrorEvent(RequestAction_PlayerRaise, param.PlayerID, err, payload.TableGame.Table)
		return
	}

	te.emitEvent("PlayerRaise", param.PlayerID, payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerCall(payload Payload) {
	playerID := payload.Param.(string)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, playerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCall, playerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Call(); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCall, playerID, err, payload.TableGame.Table)
		return
	}
	te.emitEvent("PlayerCall", playerID, payload.TableGame.Table)

	if err := te.autoNextRound(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCall, playerID, err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) handlePlayerAllin(payload Payload) {
	playerID := payload.Param.(string)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, playerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerAllin, playerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Allin(); err != nil {
		te.emitErrorEvent(RequestAction_PlayerAllin, playerID, err, payload.TableGame.Table)
		return
	}
	te.emitEvent("PlayerAllin", playerID, payload.TableGame.Table)

	if err := te.autoNextRound(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_PlayerAllin, playerID, err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) handlePlayerCheck(payload Payload) {
	playerID := payload.Param.(string)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, playerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCheck, playerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Check(); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCheck, playerID, err, payload.TableGame.Table)
		return
	}
	te.emitEvent("PlayerCheck", playerID, payload.TableGame.Table)

	if err := te.autoNextRound(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_PlayerCheck, playerID, err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) handlePlayerFold(payload Payload) {
	playerID := payload.Param.(string)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, playerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerFold, playerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Fold(); err != nil {
		te.emitErrorEvent(RequestAction_PlayerFold, playerID, err, payload.TableGame.Table)
		return
	}
	te.emitEvent("PlayerFold", playerID, payload.TableGame.Table)

	if err := te.autoNextRound(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_PlayerFold, playerID, err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) handlePlayerPass(payload Payload) {
	playerID := payload.Param.(string)

	// validate player action
	if err := te.validatePlayerMove(payload.TableGame, playerID); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPass, playerID, err, payload.TableGame.Table)
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Pass(); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPass, playerID, err, payload.TableGame.Table)
		return
	}
	te.emitEvent("PlayerPass", playerID, payload.TableGame.Table)

	if err := te.autoNextRound(payload.TableGame); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPass, playerID, err, payload.TableGame.Table)
		return
	}
}

func (te *tableEngine) validatePlayerMove(tableGame *TableGame, playerID string) error {
	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(tableGame.Table.State.PlayerStates, tableGame.Table.State.GamePlayerIndexes, playerID)
	if gamePlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// check if player can do action
	if tableGame.Game.GetState().Status.CurrentPlayer != gamePlayerIdx {
		return ErrPlayerInvalidAction
	}

	return nil
}

func (te *tableEngine) findGamePlayerIdx(players []*TablePlayerState, gamePlayerIndexes []int, targetPlayerID string) int {
	for gamePlayerIdx, playerIdx := range gamePlayerIndexes {
		player := players[playerIdx]
		if player.PlayerID == targetPlayerID {
			return gamePlayerIdx
		}
	}
	return UnsetValue
}

func (te *tableEngine) findPlayerIdx(players []*TablePlayerState, targetPlayerID string) int {
	for idx, player := range players {
		if player.PlayerID == targetPlayerID {
			return idx
		}
	}

	return UnsetValue
}

func (te *tableEngine) autoNextRound(tableGame *TableGame) error {
	event := tableGame.Table.State.GameState.Status.CurrentEvent.Name
	round := tableGame.Table.State.GameState.Status.Round

	// round not closed yet
	if event != GameEventName(pokerface.GameEvent_RoundClosed) {
		return nil
	}

	// walk situation
	if round == GameRound_Preflop && event == GameEventName(pokerface.GameEvent_GameClosed) {
		te.settleTable(tableGame.Table)
		return nil
	}

	// auto next round situation
	for {
		if err := tableGame.Game.Next(); err != nil {
			return err
		}
		gs := tableGame.Game.GetState()
		event = gs.Status.CurrentEvent.Name

		// new round started
		if event == GameEventName(pokerface.GameEvent_RoundInitialized) {
			te.runPlayerReadiesCheck(gs.GameID, tableGame)
			te.emitEvent("Auto Next Round", "", tableGame.Table)
			return nil
		}

		if event == GameEventName(pokerface.GameEvent_GameClosed) {
			te.settleTable(tableGame.Table)
			return nil
		}
	}
}

func (te *tableEngine) startGame(tableGame *TableGame) error {
	// 啟動本手遊戲引擎 & 更新遊戲狀態
	tableGame.Game = NewGame(tableGame.Table)
	if err := tableGame.Game.Start(); err != nil {
		return err
	}

	tableGame.Table.State.Status = TableStateStatus_TableGamePlaying
	tableGame.Table.State.GameState = tableGame.Game.GetState()

	tableGame.GamePlayerPayAnte = make(map[string]*syncsaga.ReadyGroup)
	tableGame.GamePlayerReadies = make(map[string]*syncsaga.ReadyGroup)

	// Set PlayerReadies
	gameID := tableGame.Table.State.GameState.GameID
	te.runPlayerReadiesCheck(gameID, tableGame)

	te.emitEvent("startGame", "", tableGame.Table)

	return nil
}

func (te *tableEngine) runPlayerReadiesCheck(gameID string, tableGame *TableGame) {
	rg := syncsaga.NewReadyGroup(
		syncsaga.WithTimeout(1, func(rg *syncsaga.ReadyGroup) {
			// Check states
			states := rg.GetParticipantStates()
			for gamePlayerIdx, isReady := range states {
				if !isReady {
					rg.Ready(gamePlayerIdx)
				}
			}
		}),
		syncsaga.WithCompletedCallback(func(rg *syncsaga.ReadyGroup) {
			if err := tableGame.Game.ReadyForAll(); err == nil {
				delete(tableGame.GamePlayerReadies, gameID)
				te.emitEvent("ReadyForAll", "", tableGame.Table)

				gs := tableGame.Table.State.GameState
				if gs.Meta.Ante > 0 && gs.Status.CurrentEvent.Name == GameEventName(pokerface.GameEvent_Prepared) {
					te.runPlayerPayAnteCheck(gameID, tableGame)
				}
			}
		}),
	)
	for gamePlayerIdx := int64(0); gamePlayerIdx < int64(len(tableGame.Table.State.GamePlayerIndexes)); gamePlayerIdx++ {
		rg.Add(gamePlayerIdx, false)
	}
	rg.Start()
	tableGame.GamePlayerReadies[gameID] = rg
}

func (te *tableEngine) runPlayerPayAnteCheck(gameID string, tableGame *TableGame) {
	rg := syncsaga.NewReadyGroup(
		syncsaga.WithTimeout(1, func(rg *syncsaga.ReadyGroup) {
			// Check states
			states := rg.GetParticipantStates()
			for gamePlayerIdx, isReady := range states {
				if !isReady {
					rg.Ready(gamePlayerIdx)
				}
			}
		}),
		syncsaga.WithCompletedCallback(func(rg *syncsaga.ReadyGroup) {
			if err := tableGame.Game.PayAnte(); err == nil {
				delete(tableGame.GamePlayerPayAnte, gameID)
				te.emitEvent("PayAnte", "", tableGame.Table)
			}
		}),
	)
	for gamePlayerIdx := int64(0); gamePlayerIdx < int64(len(tableGame.Table.State.GamePlayerIndexes)); gamePlayerIdx++ {
		rg.Add(gamePlayerIdx, false)
	}
	rg.Start()
	tableGame.GamePlayerPayAnte[gameID] = rg
}

func (te *tableEngine) settleTable(table *Table) {
	table.SettleGameResult()
	te.emitEvent("SettleGameResult", "", table)

	table.ContinueGame()
	te.emitEvent("ContinueGame", "", table)

	if table.State.Status == TableStateStatus_TablePausing && table.State.BlindState.IsBreaking() {
		// resume game from breaking
		endAt := table.State.BlindState.LevelStates[table.State.BlindState.CurrentLevelIndex].EndAt
		_ = te.timebank.NewTaskWithDeadline(time.Unix(endAt, 0), func(isCancelled bool) {
			if isCancelled {
				return
			}

			t, _ := te.GetTable(table.ID)
			if t.State.Status != TableStateStatus_TableBalancing {
				_ = te.TableGameOpen(table.ID)
			}
			te.timebank = timebank.NewTimeBank()
		})
	} else if table.State.Status == TableStateStatus_TableGameStandby {
		// 自動開桌條件: 非 TableStateStatus_TableGamePlaying 或 非 TableStateStatus_TableBalancing
		stopOpen := table.State.Status == TableStateStatus_TableGamePlaying || table.State.Status == TableStateStatus_TableBalancing
		if !stopOpen {
			_ = te.TableGameOpen(table.ID)
		}
	}
}

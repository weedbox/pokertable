package pokertable

import (
	"strings"
	"time"

	"github.com/weedbox/pokerface"
	"github.com/weedbox/timebank"
)

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
		playerIdx := payload.TableGame.Table.FindPlayerIdx(playerID)
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

	// 啟動本手遊戲引擎
	payload.TableGame.Game = newGame(payload.TableGame.Table)
	if err := payload.TableGame.Game.Start(); err != nil {
		te.emitErrorEvent(RequestAction_StartTableGame, "", err, payload.TableGame.Table)
		return
	}

	// 更新遊戲狀態
	payload.TableGame.Table.State.Status = TableStateStatus_TableGamePlaying
	payload.TableGame.Table.State.GameState = payload.TableGame.Game.GetState()

	// Update Game State
	te.updateGameState(payload.TableGame)

	te.emitEvent("TableGameStart", "", payload.TableGame.Table)
}

func (te *tableEngine) handlePlayerReady(payload Payload) {
	playerID := payload.Param.(string)

	// find game player index
	gamePlayerIdx := payload.TableGame.Table.FindGamePlayerIdx(playerID)
	if gamePlayerIdx == UnsetValue {
		te.emitErrorEvent(RequestAction_PlayerReady, playerID, ErrPlayerNotFound, payload.TableGame.Table)
		return
	}

	// valid action
	if !payload.TableGame.Game.GetState().HasAction(gamePlayerIdx, Action_Ready) {
		te.emitErrorEvent(RequestAction_PlayerReady, playerID, ErrInvalidReadyAction, payload.TableGame.Table)
		return
	}

	// do logic
	payload.TableGame.GameReadyGroup.Ready(int64(gamePlayerIdx))
}

func (te *tableEngine) handlePlayerPay(payload Payload) {
	param := payload.Param.(PlayerPayParam)

	// find game player index
	gamePlayerIdx := payload.TableGame.Table.FindGamePlayerIdx(param.PlayerID)
	if gamePlayerIdx == UnsetValue {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, ErrPlayerNotFound, payload.TableGame.Table)
		return
	}

	// valid action
	if !payload.TableGame.Game.GetState().HasAction(gamePlayerIdx, Action_Pay) {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, ErrInvalidPayAnteAction, payload.TableGame.Table)
		return
	}

	if !payload.TableGame.Game.GetState().HasAction(gamePlayerIdx, Action_Pay) {
		te.emitErrorEvent(RequestAction_PlayerReady, param.PlayerID, ErrInvalidPayAnteAction, payload.TableGame.Table)
		return
	}

	// do logic
	switch payload.TableGame.Game.GetState().Status.CurrentEvent {
	case gameEvent(pokerface.GameEvent_AnteRequested):
		fallthrough
	case gameEvent(pokerface.GameEvent_BlindsRequested):
		payload.TableGame.GameReadyGroup.Ready(int64(gamePlayerIdx))
		return
	}

	// do action
	if err := payload.TableGame.Game.GetCurrentPlayer().Pay(param.Chips); err != nil {
		te.emitErrorEvent(RequestAction_PlayerPay, param.PlayerID, err, payload.TableGame.Table)
		return
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
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

	te.updateGameState(payload.TableGame)
}

func (te *tableEngine) validatePlayerMove(tableGame *TableGame, playerID string) error {
	// find game player index
	gamePlayerIdx := tableGame.Table.FindGamePlayerIdx(playerID)
	if gamePlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// check if player can do action
	if tableGame.Game.GetState().Status.CurrentPlayer != gamePlayerIdx {
		return ErrPlayerInvalidAction
	}

	return nil
}

func (te *tableEngine) settleTableGame(tableGame *TableGame) {
	table := tableGame.Table
	table.settleTableGameResult()
	te.emitEvent("settleTableGameResult", "", table)

	table.ContinueGame()
	te.emitEvent("ContinueGame", "", table)

	// reset ready group
	tableGame.GameReadyGroup.Stop()
	tableGame.GameReadyGroup.ResetParticipants()

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
			time.Sleep(300 * time.Millisecond) // TODO: for testing only, should remove this
			_ = te.TableGameOpen(table.ID)
		}
	}
}

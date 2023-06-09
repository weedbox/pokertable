package pokertable

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

var (
	ErrInvalidCreateTableSetting = errors.New("invalid create table setting")
	ErrPlayerNotFound            = errors.New("player not found")
	ErrNoEmptySeats              = errors.New("no empty seats available")
	ErrPlayerInvalidAction       = errors.New("player invalid action")
)

type TableEngine interface {
	CreateTable(TableSetting) (Table, error)                 // 建立桌
	CloseTable(Table, TableStateStatus) Table                // 關閉桌
	StartGame(Table) (Table, error)                          // 開打遊戲
	GameOpen(table Table) (Table, error)                     // 開下一輪遊戲
	PlayerJoin(Table, JoinPlayer) (Table, error)             // 玩家入桌 (報名或補碼)
	PlayerRedeemChips(Table, JoinPlayer) (Table, error)      // 增購籌碼
	PlayersLeave(Table, []string) Table                      // 玩家們離桌
	PlayerReady(Table, string) (Table, error)                // 玩家準備動作完成
	PlayerWager(Table, string, string, int64) (Table, error) // 玩家下注
}

func NewTableEngine(gameEngine GameEngine) TableEngine {
	return &tableEngine{
		gameEngine: gameEngine,
	}
}

type tableEngine struct {
	gameEngine GameEngine
}

func (engine *tableEngine) CreateTable(tableSetting TableSetting) (Table, error) {
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
		Rankings:               make([]int, 0),
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

func (engine *tableEngine) CloseTable(table Table, status TableStateStatus) Table {
	table.State.Status = status
	table.Update()
	return table
}

func (engine *tableEngine) StartGame(table Table) (Table, error) {
	// 初始化桌 & 開局
	table = engine.TableInit(table)
	return engine.GameOpen(table)
}

/*
	PlayerJoin 玩家入桌
	  - 適用時機:
	    - 報名入桌
		- 補碼入桌
*/
func (engine *tableEngine) PlayerJoin(table Table, joinPlayer JoinPlayer) (Table, error) {
	// find player index in PlayerStates
	targetPlayerIdx := engine.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)

	// do logic
	if targetPlayerIdx == UnsetValue {
		if len(table.State.PlayerStates) == table.Meta.CompetitionMeta.TableMaxSeatCount {
			return table, ErrNoEmptySeats
		}

		// BuyIn
		player := TablePlayerState{
			PlayerID:          joinPlayer.PlayerID,
			SeatIndex:         UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          joinPlayer.RedeemChips,
		}
		table.State.PlayerStates = append(table.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(table.State.PlayerStates) - 1
		seatIdx := RandomSeatIndex(table.State.PlayerSeatMap)
		table.State.PlayerSeatMap[seatIdx] = newPlayerIdx
		table.State.PlayerStates[newPlayerIdx].SeatIndex = seatIdx
		table.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		table.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(table.State.PlayerStates[targetPlayerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
		table.State.PlayerStates[targetPlayerIdx].Bankroll += joinPlayer.RedeemChips
		table.State.PlayerStates[targetPlayerIdx].IsParticipated = true
	}

	return table, nil
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機:
	    - 增購
*/
func (engine *tableEngine) PlayerRedeemChips(table Table, joinPlayer JoinPlayer) (Table, error) {
	// find player index in PlayerStates
	playerIdx := engine.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return table, ErrPlayerNotFound
	}

	// do logic
	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	if table.State.PlayerStates[playerIdx].Bankroll == 0 {
		table.State.PlayerStates[playerIdx].IsBetweenDealerBB = IsBetweenDealerBB(table.State.PlayerStates[playerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
	}
	table.State.PlayerStates[playerIdx].Bankroll += joinPlayer.RedeemChips

	return table, nil
}

/*
	PlayerLeave 玩家們離開桌次
	  - 適用時機:
	    - CT 退桌 (玩家有籌碼)
		- CT 放棄補碼 (玩家沒有籌碼)
	    - CT/MTT 斷線且補碼中時(視為淘汰離開)
	    - CASH 離開 (準備結算)
		- CT/MTT 停止買入後被淘汰
*/
func (engine *tableEngine) PlayersLeave(table Table, playerIDs []string) Table {
	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := engine.findPlayerIdx(table.State.PlayerStates, playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return table
	}

	// do logic
	// set leave PlayerIdx int seatMap to UnsetValue
	leavePlayerIDMap := make(map[string]interface{})
	for _, leavePlayerIdx := range leavePlayerIndexes {
		leavePlayer := table.State.PlayerStates[leavePlayerIdx]
		leavePlayerIDMap[leavePlayer.PlayerID] = struct{}{}
		table.State.PlayerSeatMap[leavePlayer.SeatIndex] = UnsetValue
	}

	// delete target players in PlayerStates
	table.State.PlayerStates = funk.Filter(table.State.PlayerStates, func(player *TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*TablePlayerState)

	// update current PlayerSeatMap player indexes in PlayerSeatMap
	for newPlayerIdx, player := range table.State.PlayerStates {
		table.State.PlayerSeatMap[player.SeatIndex] = newPlayerIdx
	}

	return table
}

func (engine *tableEngine) PlayerReady(table Table, playerID string) (Table, error) {
	// find playing player index
	playingPlayerIdx := engine.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return table, ErrPlayerNotFound
	}

	// do ready
	gameState, err := engine.gameEngine.PlayerReady(playingPlayerIdx)
	if err != nil {
		fmt.Printf("[tableEngine#PlayerReady] [%s] %s ready error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	fmt.Printf("[tableEngine#PlayerReady] [%s] %s is ready. CurrentEvent: %s\n", gameState.Status.Round, playerID, gameState.Status.CurrentEvent.Name)
	table.State.GameState = gameState

	// auto pay ante, sb, bb
	if gameState.Status.CurrentEvent.Name == engine.getGameStateEventName(pokerface.GameEvent_Prepared) {
		// auto pay ante
		gameState, err := engine.gameEngine.PayAnte()
		if err != nil {
			return table, err
		}
		table.State.GameState = gameState

		if gameState.Status.Round == GameRound_Preflod {
			// auto pay sb, bb
			gameState, err := engine.gameEngine.PaySB_BB()
			if err != nil {
				return table, err
			}
			table.State.GameState = gameState
		}
	} else if gameState.Status.CurrentEvent.Name == engine.getGameStateEventName(pokerface.GameEvent_RoundInitialized) {
		if gameState.Status.Round == GameRound_Preflod {
			// auto pay sb, bb
			gameState, err := engine.gameEngine.PaySB_BB()
			if err != nil {
				return table, err
			}
			table.State.GameState = gameState

			// auto ready for all players
			gameState, err = engine.gameEngine.AllPlayersReady()
			if err != nil {
				return table, err
			}
			table.State.GameState = gameState
		}
	}

	return table, nil
}

func (engine *tableEngine) PlayerWager(table Table, playerID string, action string, chips int64) (Table, error) {
	// find playing player index
	playingPlayerIdx := engine.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return table, ErrPlayerNotFound
	}

	// check if player can do action
	if engine.gameEngine.GameState().Status.CurrentPlayer != playingPlayerIdx {
		return table, ErrPlayerInvalidAction
	}

	// do action
	gameState, err := engine.gameEngine.PlayerWager(action, chips)
	if err != nil {
		if action == WagerAction_Bet || action == WagerAction_Raise {
			fmt.Printf("[tableEngine#PlayerWager] [%s] %s %s(%d) error: %+v\n", gameState.Status.Round, playerID, action, chips, err)
		} else {
			fmt.Printf("[tableEngine#PlayerWager] [%s] %s %s error: %+v\n", gameState.Status.Round, playerID, action, err)
		}
		return table, err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := engine.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	if action == WagerAction_Bet || action == WagerAction_Raise {
		fmt.Printf("[tableEngine#PlayerWager] [%s] %s(%+v) %s(%d)\n", gameState.Status.Round, playerID, positions, action, chips)
	} else {
		fmt.Printf("[tableEngine#PlayerWager] [%s] %s(%+v) %s\n", gameState.Status.Round, playerID, positions, action)
	}
	table.State.GameState = gameState

	// Next Move
	if table.State.GameState.Status.CurrentEvent.Name == engine.getGameStateEventName(pokerface.GameEvent_RoundClosed) {
		gameState, err := engine.gameEngine.NextRound()
		if err != nil {
			return table, err
		}
		table.State.GameState = gameState

		if table.State.GameState.Status.CurrentEvent.Name == engine.getGameStateEventName(pokerface.GameEvent_GameClosed) {
			fmt.Println("[tableEngine#PlayerWager] game closed")
			table = engine.TableSettlement(table)
			engine.debugPrintGameStateResult(table) // TODO: test only, remove it later on
		}
	}

	return table, nil
}

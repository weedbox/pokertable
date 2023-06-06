package pokertable

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokermodel"
	"github.com/weedbox/pokertable/blind"
	"github.com/weedbox/pokertable/model"
	"github.com/weedbox/pokertable/position"
	"github.com/weedbox/pokertable/util"
)

var (
	ErrInvalidCreateTableSetting = errors.New("invalid create table setting")
	ErrPlayerNotFound            = errors.New("player not found")
	ErrNoEmptySeats              = errors.New("no empty seats available")
	ErrPlayerInvalidAction       = errors.New("player invalid action")
)

type TableEngine interface {
	CreateTable(model.TableSetting) (pokermodel.Table, error)                       // 建立桌
	CloseTable(pokermodel.Table, pokermodel.TableStateStatus) pokermodel.Table      // 關閉桌
	StartGame(pokermodel.Table) (pokermodel.Table, error)                           // 開打遊戲
	GameOpen(table pokermodel.Table) (pokermodel.Table, error)                      // 開下一輪遊戲
	PlayerJoin(pokermodel.Table, model.JoinPlayer) (pokermodel.Table, error)        // 玩家入桌 (報名或補碼)
	PlayerRedeemChips(pokermodel.Table, model.JoinPlayer) (pokermodel.Table, error) // 增購籌碼
	PlayersLeave(pokermodel.Table, []string) pokermodel.Table                       // 玩家們離桌
	PlayerReady(pokermodel.Table, string) (pokermodel.Table, error)                 // 玩家準備動作完成
	PlayerWager(pokermodel.Table, string, string, int64) (pokermodel.Table, error)  // 玩家下注
}

func NewTableEngine(gameEngine GameEngine) TableEngine {
	return &tableEngine{
		position:   position.NewPosition(),
		blind:      blind.NewBlind(),
		gameEngine: gameEngine,
	}
}

type tableEngine struct {
	position   position.Position
	blind      blind.Blind
	gameEngine GameEngine
}

func (engine *tableEngine) CreateTable(tableSetting model.TableSetting) (pokermodel.Table, error) {
	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.CompetitionMeta.TableMaxSeatCount {
		return pokermodel.Table{}, ErrInvalidCreateTableSetting
	}

	meta := pokermodel.TableMeta{
		ShortID:         tableSetting.ShortID,
		Code:            tableSetting.Code,
		Name:            tableSetting.Name,
		InvitationCode:  tableSetting.InvitationCode,
		CompetitionMeta: tableSetting.CompetitionMeta,
	}

	finalBuyInLevelIdx := util.UnsetValue
	if tableSetting.CompetitionMeta.Blind.FinalBuyInLevel != util.UnsetValue {
		for idx, blindLevel := range tableSetting.CompetitionMeta.Blind.Levels {
			if blindLevel.Level == tableSetting.CompetitionMeta.Blind.FinalBuyInLevel {
				finalBuyInLevelIdx = idx
				break
			}
		}
	}

	blindState := pokermodel.TableBlindState{
		FinalBuyInLevelIndex: finalBuyInLevelIdx,
		InitialLevel:         tableSetting.BlindInitialLevel,
		CurrentLevelIndex:    util.UnsetValue,
		LevelStates: funk.Map(tableSetting.CompetitionMeta.Blind.Levels, func(blindLevel pokermodel.BlindLevel) *pokermodel.TableBlindLevelState {
			return &pokermodel.TableBlindLevelState{
				BlindLevel: pokermodel.BlindLevel{
					Level:        blindLevel.Level,
					SBChips:      blindLevel.SBChips,
					BBChips:      blindLevel.BBChips,
					AnteChips:    blindLevel.AnteChips,
					DurationMins: blindLevel.DurationMins,
				},
				LevelEndAt: util.UnsetValue,
			}
		}).([]*pokermodel.TableBlindLevelState),
	}

	state := pokermodel.TableState{
		GameCount:              0,
		StartGameAt:            util.UnsetValue,
		BlindState:             &blindState,
		CurrentDealerSeatIndex: util.UnsetValue,
		CurrentBBSeatIndex:     util.UnsetValue,
		PlayerSeatMap:          engine.position.NewDefaultSeatMap(tableSetting.CompetitionMeta.TableMaxSeatCount),
		PlayerStates:           make([]*pokermodel.TablePlayerState, 0),
		PlayingPlayerIndexes:   make([]int, 0),
		Status:                 pokermodel.TableStateStatus_TableGameCreated,
		Rankings:               make([]int, 0),
	}

	// handle auto join players
	if len(tableSetting.JoinPlayers) > 0 {
		// auto join players
		state.PlayerStates = funk.Map(tableSetting.JoinPlayers, func(p model.JoinPlayer) *pokermodel.TablePlayerState {
			return &pokermodel.TablePlayerState{
				PlayerID:          p.PlayerID,
				SeatIndex:         util.UnsetValue,
				Positions:         []string{util.Position_Unknown},
				IsParticipated:    true,
				IsBetweenDealerBB: false,
				Bankroll:          p.RedeemChips,
			}
		}).([]*pokermodel.TablePlayerState)

		// update seats
		for playerIdx := 0; playerIdx < len(state.PlayerStates); playerIdx++ {
			seatIdx := engine.position.RandomSeatIndex(state.PlayerSeatMap)
			state.PlayerSeatMap[seatIdx] = playerIdx
			state.PlayerStates[playerIdx].SeatIndex = seatIdx
		}
	}

	return pokermodel.Table{
		ID:       uuid.New().String(),
		Meta:     meta,
		State:    &state,
		UpdateAt: time.Now().Unix(),
	}, nil
}

func (engine *tableEngine) CloseTable(table pokermodel.Table, status pokermodel.TableStateStatus) pokermodel.Table {
	table.State.Status = status
	table.Update()
	return table
}

func (engine *tableEngine) StartGame(table pokermodel.Table) (pokermodel.Table, error) {
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
func (engine *tableEngine) PlayerJoin(table pokermodel.Table, joinPlayer model.JoinPlayer) (pokermodel.Table, error) {
	// find player index in PlayerStates
	targetPlayerIdx := engine.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)

	// do logic
	if targetPlayerIdx == util.UnsetValue {
		if len(table.State.PlayerStates) == table.Meta.CompetitionMeta.TableMaxSeatCount {
			return table, ErrNoEmptySeats
		}

		// BuyIn
		player := pokermodel.TablePlayerState{
			PlayerID:          joinPlayer.PlayerID,
			SeatIndex:         util.UnsetValue,
			Positions:         []string{util.Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          joinPlayer.RedeemChips,
		}
		table.State.PlayerStates = append(table.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(table.State.PlayerStates) - 1
		seatIdx := engine.position.RandomSeatIndex(table.State.PlayerSeatMap)
		table.State.PlayerSeatMap[seatIdx] = newPlayerIdx
		table.State.PlayerStates[newPlayerIdx].SeatIndex = seatIdx
		table.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = engine.position.IsBetweenDealerBB(seatIdx, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		table.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = engine.position.IsBetweenDealerBB(table.State.PlayerStates[targetPlayerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
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
func (engine *tableEngine) PlayerRedeemChips(table pokermodel.Table, joinPlayer model.JoinPlayer) (pokermodel.Table, error) {
	// find player index in PlayerStates
	playerIdx := engine.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)
	if playerIdx == util.UnsetValue {
		return table, ErrPlayerNotFound
	}

	// do logic
	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	if table.State.PlayerStates[playerIdx].Bankroll == 0 {
		table.State.PlayerStates[playerIdx].IsBetweenDealerBB = engine.position.IsBetweenDealerBB(table.State.PlayerStates[playerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
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
func (engine *tableEngine) PlayersLeave(table pokermodel.Table, playerIDs []string) pokermodel.Table {
	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := engine.findPlayerIdx(table.State.PlayerStates, playerID)
		if playerIdx != util.UnsetValue {
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
		table.State.PlayerSeatMap[leavePlayer.SeatIndex] = util.UnsetValue
	}

	// delete target players in PlayerStates
	table.State.PlayerStates = funk.Filter(table.State.PlayerStates, func(player *pokermodel.TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*pokermodel.TablePlayerState)

	// update current PlayerSeatMap player indexes in PlayerSeatMap
	for newPlayerIdx, player := range table.State.PlayerStates {
		table.State.PlayerSeatMap[player.SeatIndex] = newPlayerIdx
	}

	return table
}

func (engine *tableEngine) PlayerReady(table pokermodel.Table, playerID string) (pokermodel.Table, error) {
	// find playing player index
	playingPlayerIdx := engine.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == util.UnsetValue {
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

		if gameState.Status.Round == util.GameRound_Preflod {
			// auto pay sb, bb
			gameState, err := engine.gameEngine.PaySB_BB()
			if err != nil {
				return table, err
			}
			table.State.GameState = gameState
		}
	} else if gameState.Status.CurrentEvent.Name == engine.getGameStateEventName(pokerface.GameEvent_RoundInitialized) {
		if gameState.Status.Round == util.GameRound_Preflod {
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

func (engine *tableEngine) PlayerWager(table pokermodel.Table, playerID string, action string, chips int64) (pokermodel.Table, error) {
	// find playing player index
	playingPlayerIdx := engine.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == util.UnsetValue {
		return table, ErrPlayerNotFound
	}

	// check if player can do action
	if engine.gameEngine.GameState().Status.CurrentPlayer != playingPlayerIdx {
		return table, ErrPlayerInvalidAction
	}

	// do action
	gameState, err := engine.gameEngine.PlayerWager(action, chips)
	if err != nil {
		if action == util.WagerAction_Bet || action == util.WagerAction_Raise {
			fmt.Printf("[tableEngine#PlayerWager] [%s] %s %s(%d) error: %+v\n", gameState.Status.Round, playerID, action, chips, err)
		} else {
			fmt.Printf("[tableEngine#PlayerWager] [%s] %s %s error: %+v\n", gameState.Status.Round, playerID, action, err)
		}
		return table, err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := engine.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != util.UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	if action == util.WagerAction_Bet || action == util.WagerAction_Raise {
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

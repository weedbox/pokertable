package pokertable

import (
	"errors"
	"fmt"
	"time"

	"github.com/weedbox/pokerface"

	"github.com/google/uuid"
)

var (
	ErrTableNotFound             = errors.New("table not found")
	ErrInvalidCreateTableSetting = errors.New("invalid create table setting")
	ErrPlayerNotFound            = errors.New("player not found")
	ErrNoEmptySeats              = errors.New("no empty seats available")
	ErrPlayerInvalidAction       = errors.New("player invalid action")
)

type TableEngine interface {
	// Table Actions
	OnTableUpdated(fn func(*Table)) error                     // 桌次更新監聽器
	GetTable(tableID string) (*Table, error)                  // 取得桌次
	CreateTable(tableSetting TableSetting) (*Table, error)    // 建立桌
	CloseTable(tableID string, status TableStateStatus) error // 關閉桌
	StartGame(tableID string) error                           // 開打遊戲
	GameOpen(tableID string) error                            // 開下一輪遊戲

	// Player Actions
	// Player Table Actions
	PlayerJoin(tableID string, joinPlayer JoinPlayer) error        // 玩家入桌 (報名或補碼)
	PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error // 增購籌碼
	PlayersLeave(tableID string, playerIDs []string) error         // 玩家們離桌

	// Player Game Actions
	PlayerReady(tableID, playerID string) error                  // 玩家準備動作完成
	PlayerPay(tableID, playerID string, chips int64) error       // 玩家付籌碼
	PlayerPayAnte(tableID, playerID string) error                // 玩家付前注
	PlayerPaySB(tableID, playerID string) error                  // 玩家付大盲
	PlayerPayBB(tableID, playerID string) error                  // 玩家付小盲
	PlayerBet(tableID, playerID string, chips int64) error       // 玩家下注
	PlayerRaise(tableID, playerID string, chipLevel int64) error // 玩家加注
	PlayerCall(tableID, playerID string) error                   // 玩家跟注
	PlayerAllin(tableID, playerID string) error                  // 玩家全下
	PlayerCheck(tableID, playerID string) error                  // 玩家過牌
	PlayerFold(tableID, playerID string) error                   // 玩家棄牌
}

func NewTableEngine() TableEngine {
	return &tableEngine{
		tableGameMap: make(map[string]*TableGame),
	}
}

type TableGame struct {
	Table *Table
	Game  pokerface.Game
}

type tableEngine struct {
	onTableUpdated func(*Table)
	tableGameMap   map[string]*TableGame
}

func (te *tableEngine) EmitEvent(table *Table) {
	table.RefreshUpdateAt()
	te.onTableUpdated(table)
}

func (te *tableEngine) OnTableUpdated(fn func(*Table)) error {
	te.onTableUpdated = fn
	return nil
}

func (te *tableEngine) GetTable(tableID string) (*Table, error) {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return nil, ErrTableNotFound
	}
	return tableGame.Table, nil
}

func (te *tableEngine) CreateTable(tableSetting TableSetting) (*Table, error) {
	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.CompetitionMeta.TableMaxSeatCount {
		return nil, ErrInvalidCreateTableSetting
	}

	// create table instance
	table := &Table{
		ID:       uuid.New().String(),
		UpdateAt: time.Now().Unix(),
	}
	table.ConfigureWithSetting(tableSetting)
	if len(tableSetting.JoinPlayers) > 0 {
		te.EmitEvent(table)
	}

	// update tableGameMap
	te.tableGameMap[table.ID] = &TableGame{Table: table}

	return table, nil
}

func (te *tableEngine) CloseTable(tableID string, status TableStateStatus) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	tableGame.Table.State.Status = status

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) StartGame(tableID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// 初始化桌 & 開局
	tableGame.Table.State.StartGameAt = time.Now().Unix()
	tableGame.Table.ActivateBlindState()
	tableGame.Table.GameOpen()

	// 啟動本手遊戲引擎 & 更新遊戲狀態
	tableGame.Game = NewGame(tableGame.Table)
	if err := tableGame.Game.Start(); err != nil {
		return err
	}
	tableGame.Table.State.GameState = tableGame.Game.GetState()

	te.EmitEvent(tableGame.Table)
	return nil
}

// GameOpen 開始本手遊戲
func (te *tableEngine) GameOpen(tableID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	tableGame.Table.GameOpen()

	// 啟動本手遊戲引擎 & 更新遊戲狀態
	tableGame.Game = NewGame(tableGame.Table)
	if err := tableGame.Game.Start(); err != nil {
		return err
	}
	tableGame.Table.State.GameState = tableGame.Game.GetState()

	te.EmitEvent(tableGame.Table)
	return nil
}

/*
	PlayerJoin 玩家入桌
	  - 適用時機:
	    - 報名入桌
		- 補碼入桌
*/
func (te *tableEngine) PlayerJoin(tableID string, joinPlayer JoinPlayer) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	if err := tableGame.Table.PlayerJoin(joinPlayer.PlayerID, joinPlayer.RedeemChips); err != nil {
		return err
	}

	te.EmitEvent(tableGame.Table)
	return nil
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機:
	    - 增購
*/
func (te *tableEngine) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find player index in PlayerStates
	playerIdx := tableGame.Table.findPlayerIdx(joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	tableGame.Table.PlayerRedeemChips(playerIdx, joinPlayer.RedeemChips)

	te.EmitEvent(tableGame.Table)
	return nil
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
func (te *tableEngine) PlayersLeave(tableID string, playerIDs []string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return nil
	}

	tableGame.Table.PlayersLeave(leavePlayerIndexes)

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerReady(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(tableGame.Table.State.PlayerStates, tableGame.Table.State.GamePlayerIndexes, playerID)
	if gamePlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// do ready
	if err := tableGame.Game.Ready(gamePlayerIdx); err != nil {
		fmt.Printf("[tableEngine#PlayerReady] [%s] %s ready error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}
	fmt.Printf("[tableEngine#PlayerReady] [%s] %s is ready. CurrentEvent: %s\n", tableGame.Game.GetState().Status.Round, playerID, tableGame.Game.GetState().Status.CurrentEvent.Name)

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerPay(tableID, playerID string, chips int64) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Pay(chips); err != nil {
		fmt.Printf("[tableEngine#PlayerPay] [%s] %s pay(%d) error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, chips, err)
		return err
	}
	fmt.Printf("[tableEngine#PlayerPay] dealer receive %d.\n", chips)

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerPayAnte(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	err := tableGame.Game.PayAnte()
	if err != nil {
		fmt.Printf("[tableEngine#PlayerPay] [%s] %s pay ante error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	fmt.Printf("[tableEngine#PlayerPayAnte] dealer receive ante from all players.\n")

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerPaySB(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Pay(tableGame.Game.GetState().Meta.Blind.SB); err != nil {
		fmt.Printf("[tableEngine#PlayerPaySB] [%s] %s pay sb error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	fmt.Printf("[tableEngine#PlayerPaySB] dealer receive sb.\n")

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerPayBB(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Pay(tableGame.Game.GetState().Meta.Blind.BB); err != nil {
		fmt.Printf("[tableEngine#PlayerPayBB] [%s] %s pay bb error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	fmt.Printf("[tableEngine#PlayerPayBB] dealer receive bb.\n")

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerBet(tableID, playerID string, chips int64) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Bet(chips); err != nil {
		fmt.Printf("[tableEngine#PlayerBet] [%s] %s bet(%d) error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, chips, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerBet] [%s] %s(%+v) bet(%d)\n", tableGame.Game.GetState().Status.Round, playerID, positions, chips)

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Raise(chipLevel); err != nil {
		fmt.Printf("[tableEngine#PlayerRaise] [%s] %s raise(%d) error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, chipLevel, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerRaise] [%s] %s(%+v) raise(%d)\n", tableGame.Game.GetState().Status.Round, playerID, positions, chipLevel)

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerCall(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Call(); err != nil {
		fmt.Printf("[tableEngine#PlayerCall] [%s] %s call error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerCall] [%s] %s(%+v) call\n", tableGame.Game.GetState().Status.Round, playerID, positions)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerAllin(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Allin(); err != nil {
		fmt.Printf("[tableEngine#PlayerAllin] [%s] %s allin error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerAllin] [%s] %s(%+v) allin\n", tableGame.Game.GetState().Status.Round, playerID, positions)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerCheck(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Check(); err != nil {
		fmt.Printf("[tableEngine#PlayerCheck] [%s] %s check error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerCheck] [%s] %s(%+v) check\n", tableGame.Game.GetState().Status.Round, playerID, positions)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	te.EmitEvent(tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerFold(tableID, playerID string) error {
	tableGame, exist := te.tableGameMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.Fold(); err != nil {
		fmt.Printf("[tableEngine#PlayerFold] [%s] %s fold error: %+v\n", tableGame.Game.GetState().Status.Round, playerID, err)
		return err
	}

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(tableGame.Table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = tableGame.Table.State.PlayerStates[playerIdx].Positions
	}
	fmt.Printf("[tableEngine#PlayerFold] [%s] %s(%+v) fold\n", tableGame.Game.GetState().Status.Round, playerID, positions)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	te.EmitEvent(tableGame.Table)
	return nil
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
	if tableGame.Table.State.GameState.Status.CurrentEvent.Name == GameEventName(pokerface.GameEvent_RoundClosed) {
		if err := tableGame.Game.Next(); err != nil {
			fmt.Printf("[tableEngine#autoNextRound] entering next round error: %+v\n", err)
			return err
		}

		fmt.Printf("[tableEngine#autoNextRound] entering %s\n", tableGame.Game.GetState().Status.Round)

		if tableGame.Table.State.GameState.Status.CurrentEvent.Name == GameEventName(pokerface.GameEvent_GameClosed) {
			tableGame.Table.Settlement()

			if tableGame.Table.State.Status == TableStateStatus_TableGameMatchOpen {
				// auto start next game
				te.GameOpen(tableGame.Table.ID)
			}
		}
	}
	return nil
}

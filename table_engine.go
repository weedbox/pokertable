package pokertable

import (
	"errors"
	"time"

	"github.com/weedbox/pokerface"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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

func NewTableEngine(logLevel uint32) TableEngine {
	logger := logrus.New()
	logger.SetLevel(logrus.Level(logLevel))
	return &tableEngine{
		logger:     logger,
		gameEngine: NewGameEngine(),
		tableMap:   make(map[string]*Table),
	}
}

type tableEngine struct {
	logger         *logrus.Logger
	gameEngine     *GameEngine
	onTableUpdated func(*Table)
	tableMap       map[string]*Table
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
	table, exist := te.tableMap[tableID]
	if !exist {
		return nil, ErrTableNotFound
	}
	return table, nil
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

	// update tableMap
	te.tableMap[table.ID] = table

	return table, nil
}

func (te *tableEngine) CloseTable(tableID string, status TableStateStatus) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	table.State.Status = status

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) StartGame(tableID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// 初始化桌 & 開局
	table.State.StartGameAt = time.Now().Unix()
	table.ActivateBlindState()
	if err := table.GameOpen(te.gameEngine); err != nil {
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	te.EmitEvent(table)
	return nil
}

// GameOpen 開始本手遊戲
func (te *tableEngine) GameOpen(tableID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	if err := table.GameOpen(te.gameEngine); err != nil {
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	te.EmitEvent(table)
	return nil
}

/*
	PlayerJoin 玩家入桌
	  - 適用時機:
	    - 報名入桌
		- 補碼入桌
*/
func (te *tableEngine) PlayerJoin(tableID string, joinPlayer JoinPlayer) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	if err := table.PlayerJoin(joinPlayer.PlayerID, joinPlayer.RedeemChips); err != nil {
		return err
	}

	te.EmitEvent(table)
	return nil
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機:
	    - 增購
*/
func (te *tableEngine) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find player index in PlayerStates
	playerIdx := table.findPlayerIdx(joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	table.PlayerRedeemChips(playerIdx, joinPlayer.RedeemChips)

	te.EmitEvent(table)
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
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return nil
	}

	table.PlayersLeave(leavePlayerIndexes)

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerReady(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find playing player index
	playingPlayerIdx := te.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// do ready
	err := te.gameEngine.PlayerReady(playingPlayerIdx)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerReady] [%s] %s ready error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()
	te.logger.Debugf("[tableEngine#PlayerReady] [%s] %s is ready. CurrentEvent: %s", te.gameEngine.GameState().Status.Round, playerID, te.gameEngine.GameState().Status.CurrentEvent.Name)

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerPay(tableID, playerID string, chips int64) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Pay(chips)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPay] [%s] %s pay(%d) error: %+v", te.gameEngine.GameState().Status.Round, playerID, chips, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()
	te.logger.Debugf("[tableEngine#PlayerPay] dealer receive %d.", chips)

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerPayAnte(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.PayAnte()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPay] [%s] %s pay ante error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()
	te.logger.Debug("[tableEngine#PlayerPayAnte] dealer receive ante from all players.")

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerPaySB(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.PaySB()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPaySB] [%s] %s pay sb error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()
	te.logger.Debug("[tableEngine#PlayerPaySB] dealer receive sb.")

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerPayBB(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.PayBB()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPayBB] [%s] %s pay bb error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()
	te.logger.Debug("[tableEngine#PlayerPayBB] dealer receive bb.")

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerBet(tableID, playerID string, chips int64) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Bet(chips)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerBet] [%s] %s bet(%d) error: %+v", te.gameEngine.GameState().Status.Round, playerID, chips, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerBet] [%s] %s(%+v) bet(%d)", te.gameEngine.GameState().Status.Round, playerID, positions, chips)

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Raise(chipLevel)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerRaise] [%s] %s raise(%d) error: %+v", te.gameEngine.GameState().Status.Round, playerID, chipLevel, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerRaise] [%s] %s(%+v) raise(%d)", te.gameEngine.GameState().Status.Round, playerID, positions, chipLevel)

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerCall(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Call()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerCall] [%s] %s call error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerCall] [%s] %s(%+v) call", te.gameEngine.GameState().Status.Round, playerID, positions)

	if err := te.autoNextRound(table); err != nil {
		return err
	}

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerAllin(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Allin()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerAllin] [%s] %s allin error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerAllin] [%s] %s(%+v) allin", te.gameEngine.GameState().Status.Round, playerID, positions)

	if err := te.autoNextRound(table); err != nil {
		return err
	}

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerCheck(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Check()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerCheck] [%s] %s check error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerCheck] [%s] %s(%+v) check", te.gameEngine.GameState().Status.Round, playerID, positions)

	if err := te.autoNextRound(table); err != nil {
		return err
	}

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) PlayerFold(tableID, playerID string) error {
	table, exist := te.tableMap[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(*table, playerID); err != nil {
		return err
	}

	// do action
	err := te.gameEngine.Fold()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerFold] [%s] %s fold error: %+v", te.gameEngine.GameState().Status.Round, playerID, err)
		return err
	}
	table.State.GameState = te.gameEngine.GameState()

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerFold] [%s] %s(%+v) fold", te.gameEngine.GameState().Status.Round, playerID, positions)

	if err := te.autoNextRound(table); err != nil {
		return err
	}

	te.EmitEvent(table)
	return nil
}

func (te *tableEngine) validatePlayerMove(table Table, playerID string) error {
	// find playing player index
	playingPlayerIdx := te.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// check if player can do action
	if te.gameEngine.GameState().Status.CurrentPlayer != playingPlayerIdx {
		return ErrPlayerInvalidAction
	}

	return nil
}

func (te *tableEngine) findPlayingPlayerIdx(players []*TablePlayerState, playingPlayerIndexes []int, targetPlayerID string) int {
	for idx, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		if player.PlayerID == targetPlayerID {
			return idx
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

func (te *tableEngine) autoNextRound(table *Table) error {
	if table.State.GameState.Status.CurrentEvent.Name == te.gameEngine.GameEventName(pokerface.GameEvent_RoundClosed) {
		err := te.gameEngine.NextRound()
		if err != nil {
			te.logger.Debugf("[tableEngine#autoNextRound] entering next round error: %+v", err)
			return err
		}
		table.State.GameState = te.gameEngine.GameState()

		if table.State.GameState.Status.CurrentEvent.Name == te.gameEngine.GameEventName(pokerface.GameEvent_GameClosed) {
			table.Settlement()

			if table.State.Status == TableStateStatus_TableGameMatchOpen {
				// auto start next game
				te.GameOpen(table.ID)
			}
		}
	}
	te.logger.Debug("[tableEngine#autoNextRound] entering ", te.gameEngine.GameState().Status.Round)
	return nil
}

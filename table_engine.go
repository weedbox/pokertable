package pokertable

import (
	"errors"
	"fmt"
	"time"

	"github.com/weedbox/pokerface"
	"github.com/weedbox/syncsaga"
	"github.com/weedbox/timebank"

	"github.com/google/uuid"
)

var (
	ErrTableNotFound             = errors.New("table not found")
	ErrInvalidCreateTableSetting = errors.New("invalid create table setting")
	ErrPlayerNotFound            = errors.New("player not found")
	ErrNoEmptySeats              = errors.New("no empty seats available")
	ErrPlayerInvalidAction       = errors.New("player invalid action")
	ErrCloseTable                = errors.New("table close error")
	ErrTableGameError            = errors.New("table game error")
	ErrInvalidReadyAction        = errors.New("invalid ready action")
	ErrInvalidPayAnteAction      = errors.New("invalid pay ante action")
)

type TableEvent string

const (
	TableEvent_Updated TableEvent = "table_updated"
)

type TableEngine interface {
	// Table Actions
	OnTableUpdated(fn func(*Table))                        // 桌次更新事件監聽器
	GetTable(tableID string) (*Table, error)               // 取得桌次
	CreateTable(tableSetting TableSetting) (*Table, error) // 建立桌
	BalanceTable(tableID string) error                     // 等待拆併桌中
	DeleteTable(tableID string) error                      // 刪除桌
	StartTableGame(tableID string) error                   // 開打遊戲
	TableGameOpen(tableID string) error                    // 開下一輪遊戲

	// Player Actions
	// Player Table Actions
	PlayerJoin(tableID string, joinPlayer JoinPlayer) error        // 玩家入桌 (報名或補碼)
	PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error // 增購籌碼
	PlayersLeave(tableID string, playerIDs []string) error         // 玩家們離桌

	// Player Game Actions
	PlayerReady(tableID, playerID string) error                  // 玩家準備動作完成
	PlayerPay(tableID, playerID string, chips int64) error       // 玩家付籌碼
	PlayerBet(tableID, playerID string, chips int64) error       // 玩家下注
	PlayerRaise(tableID, playerID string, chipLevel int64) error // 玩家加注
	PlayerCall(tableID, playerID string) error                   // 玩家跟注
	PlayerAllin(tableID, playerID string) error                  // 玩家全下
	PlayerCheck(tableID, playerID string) error                  // 玩家過牌
	PlayerFold(tableID, playerID string) error                   // 玩家棄牌
	PlayerPass(tableID, playerID string) error                   // 玩家 Pass
}

func NewTableEngine() TableEngine {
	return &tableEngine{
		timebank:   timebank.NewTimeBank(),
		tableGames: make(map[string]*TableGame),
	}
}

type TableGame struct {
	Table             *Table
	Game              pokerface.Game
	GamePlayerReadies map[string]*syncsaga.ReadyGroup
	GamePlayerPayAnte map[string]*syncsaga.ReadyGroup
}

type tableEngine struct {
	timebank       *timebank.TimeBank
	onTableUpdated func(*Table)
	tableGames     map[string]*TableGame
}

func (te *tableEngine) EmitEvent(eventName string, table *Table) {
	table.RefreshUpdateAt()
	fmt.Printf("[#%d][%d] Emit Event: %s\n", table.UpdateSerial, table.State.GameCount, eventName)
	// json, _ := table.GetGameStateJSON()
	// fmt.Println(fmt.Sprintf("[#%d]", table.UpdateSerial), json)

	if te.onTableUpdated != nil {
		te.onTableUpdated(table)
	}
}

func (te *tableEngine) OnTableUpdated(fn func(*Table)) {
	te.onTableUpdated = fn
}

func (te *tableEngine) GetTable(tableID string) (*Table, error) {
	tableGame, exist := te.tableGames[tableID]
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
		ID: uuid.New().String(),
	}
	table.ConfigureWithSetting(tableSetting, TableStateStatus_TableCreated)
	te.EmitEvent("CreateTable", table)

	// update tableGames
	te.tableGames[table.ID] = &TableGame{Table: table}

	return table, nil
}

/*
	BalanceTable 等待拆併桌
	  - 適用時機: 該桌次需要拆併桌時
*/
func (te *tableEngine) BalanceTable(tableID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	tableGame.Table.State.Status = TableStateStatus_TableBalancing
	return nil
}

/*
	DeleteTable 刪除桌
	  - 適用時機: 強制關閉 (Killed)、自動關閉 (AutoEnded)、正常關閉 (Closed)
*/
func (te *tableEngine) DeleteTable(tableID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	tableGame.Table.State.Status = TableStateStatus_TableClosed
	te.EmitEvent("DeleteTable", tableGame.Table)

	// update tableGames
	delete(te.tableGames, tableID)

	return nil
}

func (te *tableEngine) StartTableGame(tableID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// 初始化桌 & 開局 & 開始遊戲
	tableGame.Table.State.StartAt = time.Now().Unix()
	tableGame.Table.ActivateBlindState()
	return te.TableGameOpen(tableID)
}

func (te *tableEngine) TableGameOpen(tableID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// 開局
	tableGame.Table.OpenGame()
	te.EmitEvent("TableGameOpen", tableGame.Table)

	// 開始遊戲
	return te.startGame(tableGame)
}

/*
	PlayerJoin 玩家入桌
	  - 適用時機: 報名入桌、補碼入桌
*/
func (te *tableEngine) PlayerJoin(tableID string, joinPlayer JoinPlayer) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	if err := tableGame.Table.PlayerJoin(joinPlayer.PlayerID, joinPlayer.RedeemChips); err != nil {
		return err
	}

	te.EmitEvent("PlayerJoin", tableGame.Table)
	return nil
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機: 增購
*/
func (te *tableEngine) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find player index in PlayerStates
	playerIdx := tableGame.Table.findPlayerIdx(joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	tableGame.Table.PlayerRedeemChips(playerIdx, joinPlayer.RedeemChips)

	te.EmitEvent("PlayerRedeemChips", tableGame.Table)
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
	tableGame, exist := te.tableGames[tableID]
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

	te.EmitEvent("PlayersLeave", tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerReady(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(tableGame.Table.State.PlayerStates, tableGame.Table.State.GamePlayerIndexes, playerID)
	if gamePlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// handle ready group
	rg, exist := tableGame.GamePlayerReadies[tableGame.Game.GetState().GameID]
	if !exist {
		return ErrInvalidReadyAction
	}
	rg.Ready(int64(gamePlayerIdx))

	return nil
}

func (te *tableEngine) PlayerPay(tableID, playerID string, chips int64) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// find game player index
	gamePlayerIdx := te.findGamePlayerIdx(tableGame.Table.State.PlayerStates, tableGame.Table.State.GamePlayerIndexes, playerID)
	if gamePlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// handle ready group
	gs := tableGame.Game.GetState()
	event := gs.Status.CurrentEvent.Name
	// Pay Ante: call pay ante ready group ready
	if chips == gs.Meta.Ante && event == GameEventName(pokerface.GameEvent_Prepared) {
		rg, exist := tableGame.GamePlayerPayAnte[tableGame.Game.GetState().GameID]
		if !exist {
			return ErrInvalidPayAnteAction
		}
		rg.Ready(int64(gamePlayerIdx))
		return nil
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Pay(chips); err != nil {
		return err
	}

	// After Pay BB: run readies ready group
	if chips == gs.Meta.Blind.BB && event == GameEventName(pokerface.GameEvent_RoundInitialized) {
		te.runPlayerReadiesCheck(gs.GameID, tableGame)
	}
	te.EmitEvent("PlayerPay", tableGame.Table)

	return nil
}

func (te *tableEngine) PlayerBet(tableID, playerID string, chips int64) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Bet(chips); err != nil {
		return err
	}
	te.EmitEvent("PlayerBet", tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Raise(chipLevel); err != nil {
		return err
	}
	te.EmitEvent("PlayerRaise", tableGame.Table)
	return nil
}

func (te *tableEngine) PlayerCall(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Call(); err != nil {
		return err
	}
	te.EmitEvent("PlayerCall", tableGame.Table)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	return nil
}

func (te *tableEngine) PlayerAllin(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Allin(); err != nil {
		return err
	}
	te.EmitEvent("PlayerAllin", tableGame.Table)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	return nil
}

func (te *tableEngine) PlayerCheck(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Check(); err != nil {
		return err
	}
	te.EmitEvent("PlayerCheck", tableGame.Table)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	return nil
}

func (te *tableEngine) PlayerFold(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Fold(); err != nil {
		return err
	}
	te.EmitEvent("PlayerFold", tableGame.Table)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

	return nil
}

func (te *tableEngine) PlayerPass(tableID, playerID string) error {
	tableGame, exist := te.tableGames[tableID]
	if !exist {
		return ErrTableNotFound
	}

	// validate player action
	if err := te.validatePlayerMove(tableGame, playerID); err != nil {
		return err
	}

	// do action
	if err := tableGame.Game.GetCurrentPlayer().Pass(); err != nil {
		return err
	}
	te.EmitEvent("PlayerPass", tableGame.Table)

	if err := te.autoNextRound(tableGame); err != nil {
		return err
	}

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
			te.EmitEvent("Auto Next Round", tableGame.Table)
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

	te.EmitEvent("startGame", tableGame.Table)

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
				te.EmitEvent("ReadyForAll", tableGame.Table)

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
				te.EmitEvent("PayAnte", tableGame.Table)
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
	te.EmitEvent("SettleGameResult", table)

	table.ContinueGame()
	te.EmitEvent("ContinueGame", table)

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

package pokertable

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/weedbox/syncsaga"
	"github.com/weedbox/timebank"
)

var (
	ErrTableNoEmptySeats            = errors.New("table: no empty seats available")
	ErrTableInvalidCreateSetting    = errors.New("table: invalid create table setting")
	ErrTablePlayerNotFound          = errors.New("table: player not found")
	ErrTablePlayerInvalidGameAction = errors.New("table: player invalid game action")
	ErrTablePlayerInvalidAction     = errors.New("table: player invalid action")
	ErrTableOpenGameFailed          = errors.New("table: failed to open game")
)

type TableEngineOpt func(*tableEngine)

type TableEngine interface {
	// Events
	OnTableUpdated(fn func(*Table))                                                              // 桌次更新事件監聽器
	OnTableErrorUpdated(fn func(*Table, error))                                                  // 錯誤更新事件監聽器
	OnTableStateUpdated(fn func(string, *Table))                                                 // 桌次狀態監聽器
	OnTablePlayerStateUpdated(fn func(string, string, *TablePlayerState))                        // 桌次玩家狀態監聽器
	OnTablePlayerReserved(fn func(competitionID, tableID string, playerState *TablePlayerState)) // 桌次玩家確認座位監聽器
	OnGamePlayerActionUpdated(fn func(TablePlayerGameAction))                                    // 遊戲玩家動作更新事件監聽器

	// Table Actions
	GetTable() *Table                                      // 取得桌次
	GetGame() Game                                         // 取得遊戲引擎
	CreateTable(tableSetting TableSetting) (*Table, error) // 建立桌
	BalanceTable() error                                   // 等待拆併桌中
	PauseTable() error                                     // 暫停桌
	CloseTable() error                                     // 關閉桌
	StartTableGame() error                                 // 開打遊戲
	TableGameOpen() error                                  // 開下一輪遊戲
	UpdateBlind(level int, ante, dealer, sb, bb int64)     // 更新當前盲注資訊

	// Player Table Actions
	PlayerReserve(joinPlayer JoinPlayer) error          // 玩家確認座位
	PlayersBatchReserve(joinPlayers []JoinPlayer) error // 整桌玩家確認座位
	PlayerJoin(playerID string) error                   // 玩家入桌
	PlayerRedeemChips(joinPlayer JoinPlayer) error      // 增購籌碼
	PlayersLeave(playerIDs []string) error              // 玩家們離桌

	// Player Game Actions
	PlayerReady(playerID string) error                  // 玩家準備動作完成
	PlayerPay(playerID string, chips int64) error       // 玩家付籌碼
	PlayerBet(playerID string, chips int64) error       // 玩家下注
	PlayerRaise(playerID string, chipLevel int64) error // 玩家加注
	PlayerCall(playerID string) error                   // 玩家跟注
	PlayerAllin(playerID string) error                  // 玩家全下
	PlayerCheck(playerID string) error                  // 玩家過牌
	PlayerFold(playerID string) error                   // 玩家棄牌
	PlayerPass(playerID string) error                   // 玩家 Pass
}

type tableEngine struct {
	lock                      sync.Mutex
	options                   *TableEngineOptions
	table                     *Table
	game                      Game
	gameBackend               GameBackend
	rg                        *syncsaga.ReadyGroup
	tb                        *timebank.TimeBank
	onTableUpdated            func(*Table)
	onTableErrorUpdated       func(*Table, error)
	onTableStateUpdated       func(string, *Table)
	onTablePlayerStateUpdated func(string, string, *TablePlayerState)
	onTablePlayerReserved     func(competitionID, tableID string, playerState *TablePlayerState)
	onGamePlayerActionUpdated func(TablePlayerGameAction)
}

func NewTableEngine(options *TableEngineOptions, opts ...TableEngineOpt) TableEngine {
	callbacks := NewTableEngineCallbacks()
	te := &tableEngine{
		options:                   options,
		rg:                        syncsaga.NewReadyGroup(),
		tb:                        timebank.NewTimeBank(),
		onTableUpdated:            callbacks.OnTableUpdated,
		onTableErrorUpdated:       callbacks.OnTableErrorUpdated,
		onTableStateUpdated:       callbacks.OnTableStateUpdated,
		onTablePlayerStateUpdated: callbacks.OnTablePlayerStateUpdated,
		onTablePlayerReserved:     callbacks.OnTablePlayerReserved,
		onGamePlayerActionUpdated: callbacks.OnGamePlayerActionUpdated,
	}

	for _, opt := range opts {
		opt(te)
	}

	return te
}

func WithGameBackend(gb GameBackend) TableEngineOpt {
	return func(te *tableEngine) {
		te.gameBackend = gb
	}
}

func (te *tableEngine) OnTableUpdated(fn func(*Table)) {
	te.onTableUpdated = fn
}

func (te *tableEngine) OnTableErrorUpdated(fn func(*Table, error)) {
	te.onTableErrorUpdated = fn
}

func (te *tableEngine) OnTableStateUpdated(fn func(string, *Table)) {
	te.onTableStateUpdated = fn
}

func (te *tableEngine) OnTablePlayerStateUpdated(fn func(string, string, *TablePlayerState)) {
	te.onTablePlayerStateUpdated = fn
}

func (te *tableEngine) OnTablePlayerReserved(fn func(competitionID, tableID string, playerState *TablePlayerState)) {
	te.onTablePlayerReserved = fn
}

func (te *tableEngine) OnGamePlayerActionUpdated(fn func(TablePlayerGameAction)) {
	te.onGamePlayerActionUpdated = fn
}

func (te *tableEngine) GetTable() *Table {
	return te.table
}

func (te *tableEngine) GetGame() Game {
	return te.game
}

func (te *tableEngine) CreateTable(tableSetting TableSetting) (*Table, error) {
	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.Meta.TableMaxSeatCount {
		return nil, ErrTableInvalidCreateSetting
	}

	// create table instance
	table := &Table{
		ID: tableSetting.TableID,
	}

	// configure meta
	table.Meta = tableSetting.Meta

	// configure state
	state := TableState{
		GameCount: 0,
		StartAt:   UnsetValue,
		BlindState: &TableBlindState{
			Level:  0,
			Ante:   UnsetValue,
			Dealer: UnsetValue,
			SB:     UnsetValue,
			BB:     UnsetValue,
		},
		CurrentDealerSeat: UnsetValue,
		CurrentBBSeat:     UnsetValue,
		SeatMap:           NewDefaultSeatMap(tableSetting.Meta.TableMaxSeatCount),
		PlayerStates:      make([]*TablePlayerState, 0),
		GamePlayerIndexes: make([]int, 0),
		Status:            TableStateStatus_TableCreated,
	}
	table.State = &state
	te.table = table

	// handle auto join players
	if len(tableSetting.JoinPlayers) > 0 {
		te.PlayersBatchReserve(tableSetting.JoinPlayers)
	}

	te.emitEvent("CreateTable", "")
	te.emitTableStateEvent(TableStateEvent_Created)
	return te.table, nil
}

/*
BalanceTable 等待拆併桌
  - 適用時機: 該桌次需要拆併桌時
*/
func (te *tableEngine) BalanceTable() error {
	te.table.State.Status = TableStateStatus_TableBalancing

	te.emitEvent("BalanceTable", "")
	te.emitTableStateEvent(TableStateEvent_StatusUpdated)
	return nil
}

/*
PauseTable 暫停桌
  - 適用時機: 外部暫停自動開桌
*/
func (te *tableEngine) PauseTable() error {
	te.table.State.Status = TableStateStatus_TablePausing
	te.emitTableStateEvent(TableStateEvent_StatusUpdated)
	return nil
}

/*
CloseTable 關閉桌次
  - 適用時機: 強制關閉、逾期自動關閉、正常關閉
*/
func (te *tableEngine) CloseTable() error {
	te.table.State.Status = TableStateStatus_TableClosed

	te.emitEvent("CloseTable", "")
	te.emitTableStateEvent(TableStateEvent_StatusUpdated)
	return nil
}

func (te *tableEngine) StartTableGame() error {
	// 更新開始時間
	te.table.State.StartAt = time.Now().Unix()
	te.emitEvent("StartTableGame", "")

	//  開局
	return te.TableGameOpen()
}

func (te *tableEngine) TableGameOpen() error {
	// 開局
	newTable, err := te.openGame(te.table)

	retry := 7
	if err != nil {
		// 20 秒內嘗試重新開局，每三秒一次，共七次
		if err == ErrTableOpenGameFailed {
			reopened := false

			for i := 0; i < retry; i++ {
				time.Sleep(time.Second * 3)
				newTable, err = te.openGame(te.table)
				if err != nil {
					if err == ErrTableOpenGameFailed {
						fmt.Printf("table (%s): failed to open game. retry %d time(s)...\n", te.table.ID, i+1)
						continue
					} else {
						return err
					}
				} else {
					reopened = true
					break
				}
			}

			if !reopened {
				return err
			}
		} else {
			return err
		}
	}
	te.table = newTable
	te.emitEvent("TableGameOpen", "")

	// 啟動本手遊戲引擎
	return te.startGame()
}

func (te *tableEngine) UpdateBlind(level int, ante, dealer, sb, bb int64) {
	te.table.State.BlindState.Level = level
	te.table.State.BlindState.Ante = ante
	te.table.State.BlindState.Dealer = dealer
	te.table.State.BlindState.SB = sb
	te.table.State.BlindState.BB = bb
}

/*
PlayerReserve 玩家確認座位
  - 適用時機: 玩家帶籌碼報名或補碼
*/
func (te *tableEngine) PlayerReserve(joinPlayer JoinPlayer) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	playerID := joinPlayer.PlayerID
	redeemChips := joinPlayer.RedeemChips
	seat := joinPlayer.Seat

	// find player index in PlayerStates
	targetPlayerIdx := te.table.FindPlayerIdx(playerID)

	if targetPlayerIdx == UnsetValue {
		if len(te.table.State.PlayerStates) == te.table.Meta.TableMaxSeatCount {
			return ErrTableNoEmptySeats
		}

		// BuyIn
		player := TablePlayerState{
			PlayerID:          playerID,
			Seat:              UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    false,
			IsBetweenDealerBB: false,
			Bankroll:          redeemChips,
			IsIn:              false,
			GameStatistics:    TablePlayerGameStatistics{},
		}
		te.table.State.PlayerStates = append(te.table.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(te.table.State.PlayerStates) - 1

		// decide seat
		seatIdx := RandomSeat(te.table.State.SeatMap)
		if seat != UnsetValue && te.table.State.SeatMap[seat] == UnsetValue {
			seatIdx = seat
		}
		te.table.State.SeatMap[seatIdx] = newPlayerIdx

		playerState := te.table.State.PlayerStates[newPlayerIdx]
		playerState.Seat = seatIdx
		playerState.IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.TableMaxSeatCount, te.table.Meta.Rule)

		te.emitTablePlayerStateEvent(playerState)
		te.emitTablePlayerReservedEvent(playerState)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		playerState := te.table.State.PlayerStates[targetPlayerIdx]
		playerState.IsBetweenDealerBB = IsBetweenDealerBB(playerState.Seat, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.TableMaxSeatCount, te.table.Meta.Rule)
		playerState.Bankroll += redeemChips

		te.emitTablePlayerStateEvent(playerState)
		te.emitTablePlayerReservedEvent(playerState)
	}

	te.emitEvent("PlayerReserve", joinPlayer.PlayerID)

	return nil
}

/*
PlayersBatchReserve 多位玩家確認座位
  - 適用時機: 拆併桌整桌玩家確認座位、開桌時有預設玩家
*/
func (te *tableEngine) PlayersBatchReserve(joinPlayers []JoinPlayer) error {
	// Preparing ready group for waiting all players' join
	te.rg.Stop()
	te.rg.SetTimeoutInterval(15)
	te.rg.OnTimeout(func(rg *syncsaga.ReadyGroup) {
		// Auto Ready By Default
		states := rg.GetParticipantStates()
		for playerIdx, isReady := range states {
			if !isReady {
				fmt.Printf("[DEBUG#tableEngine#PlayersBatchReserve] table [%s] %s is auto ready\n", te.table.ID, te.table.State.PlayerStates[playerIdx].PlayerID)
				rg.Ready(playerIdx)
			}
		}
	})
	te.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		fmt.Printf("[DEBUG#tableEngine#PlayersBatchReserve] OnCompleted. Status:%s\n", te.table.State.Status)
		if te.table.State.Status == TableStateStatus_TableBalancing {
			for i := 0; i < len(te.table.State.PlayerStates); i++ {
				// 如果時間到了還沒有入座則自動入座
				if !te.table.State.PlayerStates[i].IsIn {
					te.table.State.PlayerStates[i].IsIn = true
				}
			}

			if te.table.State.GameCount <= 0 {
				// 拆併桌起新桌，時間到了自動開打
				if err := te.StartTableGame(); err != nil {
					te.onTableErrorUpdated(te.table, err)
				}
			}
		}
	})

	// reserve player
	if te.table.State.GameCount <= 0 {
		te.table.State.Status = TableStateStatus_TableBalancing
	}
	copyTable := *te.table
	for _, joinPlayer := range joinPlayers {
		if err := te.PlayerReserve(joinPlayer); err != nil {
			te.table = &copyTable
			te.rg.Stop()
			te.rg.ResetParticipants()
			return err
		}
	}

	te.rg.ResetParticipants()
	for playerIdx := range te.table.State.PlayerStates {
		if !te.table.State.PlayerStates[playerIdx].IsIn {
			// 新加入的玩家才要放到 ready group 做處理
			te.rg.Add(int64(playerIdx), false)
		}
	}

	te.rg.Start()

	te.emitEvent("PlayersBatchReserve", "")

	return nil
}

/*
PlayerJoin 玩家入桌
  - 適用時機: 玩家已經確認座位後入桌
*/
func (te *tableEngine) PlayerJoin(playerID string) error {
	playerIdx := te.table.FindPlayerIdx(playerID)
	if playerIdx == UnsetValue {
		return ErrTablePlayerNotFound
	}

	if te.table.State.PlayerStates[playerIdx].Seat == UnsetValue {
		return ErrTablePlayerInvalidAction
	}

	if te.table.State.PlayerStates[playerIdx].IsIn {
		return nil
	}

	te.table.State.PlayerStates[playerIdx].IsIn = true

	if te.table.State.Status == TableStateStatus_TableBalancing {
		// fmt.Printf("[DEBUG#tableEngine#PlayerJoin] table [%s] %s is ready", te.table.ID, te.table.State.PlayerStates[playerIdx].PlayerID)
		te.rg.Ready(int64(playerIdx))
	}

	te.emitEvent("PlayerJoin", playerID)
	return nil
}

/*
PlayerRedeemChips 增購籌碼
  - 適用時機: 增購
*/
func (te *tableEngine) PlayerRedeemChips(joinPlayer JoinPlayer) error {
	// find player index in PlayerStates
	playerIdx := te.table.FindPlayerIdx(joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return ErrTablePlayerNotFound
	}

	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	playerState := te.table.State.PlayerStates[playerIdx]
	if playerState.Bankroll == 0 {
		playerState.IsBetweenDealerBB = IsBetweenDealerBB(playerState.Seat, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.TableMaxSeatCount, te.table.Meta.Rule)
	}
	playerState.Bankroll += joinPlayer.RedeemChips

	te.emitEvent("PlayerRedeemChips", joinPlayer.PlayerID)
	te.emitTablePlayerStateEvent(playerState)
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
func (te *tableEngine) PlayersLeave(playerIDs []string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	newPlayerStates, newSeatMap, newGamePlayerIndexes := te.calcLeavePlayers(te.table.State.Status, playerIDs, te.table.State.PlayerStates, te.table.Meta.TableMaxSeatCount)
	te.table.State.PlayerStates = newPlayerStates
	te.table.State.SeatMap = newSeatMap
	te.table.State.GamePlayerIndexes = newGamePlayerIndexes

	te.emitEvent("PlayersLeave", strings.Join(playerIDs, ","))
	te.emitTableStateEvent(TableStateEvent_PlayersLeave)

	return nil
}

func (te *tableEngine) PlayerReady(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Ready(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, "ready", 0, gs.GetPlayer(gamePlayerIdx))
	}
	return err
}

func (te *tableEngine) PlayerPay(playerID string, chips int64) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Pay(gamePlayerIdx, chips)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, "pay", chips, gs.GetPlayer(gamePlayerIdx))
	}
	return err
}

func (te *tableEngine) PlayerBet(playerID string, chips int64) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Bet(gamePlayerIdx, chips)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_Bet, chips, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		if te.game.GetGameState().Status.CurrentRaiser == gamePlayerIdx {
			playerState.GameStatistics.RaiseTimes++
		}
	}
	return err
}

func (te *tableEngine) PlayerRaise(playerID string, chipLevel int64) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Raise(gamePlayerIdx, chipLevel)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_Raise, chipLevel, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		playerState.GameStatistics.RaiseTimes++
	}
	return err
}

func (te *tableEngine) PlayerCall(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	wager := int64(0)
	if te.table.State.GameState != nil && gamePlayerIdx < len(te.table.State.GameState.Players) {
		wager = te.table.State.GameState.Status.CurrentWager - te.table.State.GameState.GetPlayer(gamePlayerIdx).Wager
	}

	gs, err := te.game.Call(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_Call, wager, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		playerState.GameStatistics.CallTimes++
	}
	return err
}

func (te *tableEngine) PlayerAllin(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	wager := int64(0)
	if te.table.State.GameState != nil && gamePlayerIdx < len(te.table.State.GameState.Players) {
		wager = te.table.State.GameState.GetPlayer(gamePlayerIdx).StackSize
	}

	gs, err := te.game.Allin(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_AllIn, wager, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		if te.game.GetGameState().Status.CurrentRaiser == gamePlayerIdx {
			playerState.GameStatistics.RaiseTimes++
		}
	}
	return err
}

func (te *tableEngine) PlayerCheck(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Check(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_Check, 0, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		playerState.GameStatistics.CheckTimes++
	}
	return err
}

func (te *tableEngine) PlayerFold(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Fold(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, WagerAction_Fold, 0, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)

		playerState := te.table.State.PlayerStates[playerIdx]
		playerState.GameStatistics.ActionTimes++
		playerState.GameStatistics.IsFold = true
		playerState.GameStatistics.FoldRound = te.game.GetGameState().Status.Round
	}
	return err
}

func (te *tableEngine) PlayerPass(playerID string) error {
	te.lock.Lock()
	defer te.lock.Unlock()

	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	gs, err := te.game.Pass(gamePlayerIdx)
	if err == nil {
		playerIdx := te.table.State.GamePlayerIndexes[gamePlayerIdx]
		te.table.State.LastPlayerGameAction = te.createPlayerGameAction(playerID, playerIdx, "pass", 0, gs.GetPlayer(gamePlayerIdx))
		te.emitGamePlayerActionEvent(*te.table.State.LastPlayerGameAction)
	}
	return err
}

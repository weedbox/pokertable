package pokertable

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thoas/go-funk"
	"github.com/weedbox/syncsaga"
	"github.com/weedbox/timebank"
)

var (
	ErrTableNoEmptySeats            = errors.New("table: no empty seats available")
	ErrTableInvalidCreateSetting    = errors.New("table: invalid create table setting")
	ErrTablePlayerNotFound          = errors.New("table: player not found")
	ErrTablePlayerInvalidGameAction = errors.New("table: player invalid game action")
	ErrTablePlayerInvalidAction     = errors.New("table: player invalid action")
)

type TableEngineOpt func(*tableEngine)

type TableEngineOptions struct {
	Interval int
}

func NewTableEngineOptions() *TableEngineOptions {
	return &TableEngineOptions{
		Interval: 0, // 0 second by default
	}
}

type TableEngine interface {
	// Events
	OnTableUpdated(fn func(*Table))             // 桌次更新事件監聽器
	OnTableErrorUpdated(fn func(*Table, error)) // 錯誤更新事件監聽器

	// Table Actions
	GetTable() *Table                                      // 取得桌次
	GetGame() Game                                         // 取得遊戲引擎
	CreateTable(tableSetting TableSetting) (*Table, error) // 建立桌
	BalanceTable() error                                   // 等待拆併桌中
	CloseTable() error                                     // 關閉桌
	StartTableGame() error                                 // 開打遊戲
	TableGameOpen() error                                  // 開下一輪遊戲

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
	options             *TableEngineOptions
	table               *Table
	game                Game
	gameBackend         GameBackend
	rg                  *syncsaga.ReadyGroup
	tb                  *timebank.TimeBank
	onTableUpdated      func(*Table)
	onTableErrorUpdated func(*Table, error)
}

func NewTableEngine(options *TableEngineOptions, opts ...TableEngineOpt) TableEngine {
	te := &tableEngine{
		options:             options,
		rg:                  syncsaga.NewReadyGroup(),
		tb:                  timebank.NewTimeBank(),
		onTableUpdated:      func(t *Table) {},
		onTableErrorUpdated: func(t *Table, err error) {},
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

func (te *tableEngine) GetTable() *Table {
	return te.table
}

func (te *tableEngine) GetGame() Game {
	return te.game
}

func (te *tableEngine) CreateTable(tableSetting TableSetting) (*Table, error) {
	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.CompetitionMeta.TableMaxSeatCount {
		return nil, ErrTableInvalidCreateSetting
	}

	// create table instance
	table := &Table{
		ID: uuid.New().String(),
	}

	// configure meta
	meta := TableMeta{
		ShortID:         tableSetting.ShortID,
		Code:            tableSetting.Code,
		Name:            tableSetting.Name,
		InvitationCode:  tableSetting.InvitationCode,
		CompetitionMeta: tableSetting.CompetitionMeta,
	}
	table.Meta = meta

	// configure state
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
				BlindLevel: blindLevel,
				EndAt:      UnsetValue,
			}
		}).([]*TableBlindLevelState),
	}
	state := TableState{
		GameCount:         0,
		StartAt:           UnsetValue,
		BlindState:        &blindState,
		CurrentDealerSeat: UnsetValue,
		CurrentBBSeat:     UnsetValue,
		SeatMap:           NewDefaultSeatMap(tableSetting.CompetitionMeta.TableMaxSeatCount),
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
	return te.table, nil
}

/*
	BalanceTable 等待拆併桌
	  - 適用時機: 該桌次需要拆併桌時
*/
func (te *tableEngine) BalanceTable() error {
	te.table.State.Status = TableStateStatus_TableBalancing

	te.emitEvent("BalanceTable", "")
	return nil
}

/*
	CloseTable 關閉桌次
	  - 適用時機: 強制關閉、逾期自動關閉、正常關閉
*/
func (te *tableEngine) CloseTable() error {
	te.table.State.Status = TableStateStatus_TableClosed

	te.emitEvent("CloseTable", "")
	return nil
}

func (te *tableEngine) StartTableGame() error {
	// 更新開始時間
	te.table.State.StartAt = time.Now().Unix()

	// 初始化盲注
	for idx, levelState := range te.table.State.BlindState.LevelStates {
		if levelState.BlindLevel.Level == te.table.State.BlindState.InitialLevel {
			te.table.State.BlindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := te.table.State.StartAt
	for i := (te.table.State.BlindState.InitialLevel - 1); i < len(te.table.State.BlindState.LevelStates); i++ {
		if i == te.table.State.BlindState.InitialLevel-1 {
			te.table.State.BlindState.LevelStates[i].EndAt = blindStartAt
		} else {
			te.table.State.BlindState.LevelStates[i].EndAt = te.table.State.BlindState.LevelStates[i-1].EndAt
		}
		blindPassedSeconds := int64(te.table.State.BlindState.LevelStates[i].BlindLevel.Duration)
		te.table.State.BlindState.LevelStates[i].EndAt += blindPassedSeconds
	}

	te.emitEvent("StartTableGame", "")

	//  開局
	go te.TableGameOpen()

	return nil
}

func (te *tableEngine) TableGameOpen() error {
	// 開局
	te.openGame()

	te.emitEvent("TableGameOpen", "")

	// 啟動本手遊戲引擎
	return te.startGame()
}

/*
	PlayerReserve 玩家確認座位
	  - 適用時機: 玩家帶籌碼報名或補碼
*/
func (te *tableEngine) PlayerReserve(joinPlayer JoinPlayer) error {
	playerID := joinPlayer.PlayerID
	redeemChips := joinPlayer.RedeemChips
	seat := joinPlayer.Seat

	// find player index in PlayerStates
	targetPlayerIdx := te.table.FindPlayerIdx(playerID)

	if targetPlayerIdx == UnsetValue {
		if len(te.table.State.PlayerStates) == te.table.Meta.CompetitionMeta.TableMaxSeatCount {
			return ErrTableNoEmptySeats
		}

		// BuyIn
		player := TablePlayerState{
			PlayerID:          playerID,
			Seat:              UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          redeemChips,
			IsIn:              false,
		}
		te.table.State.PlayerStates = append(te.table.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(te.table.State.PlayerStates) - 1

		// decide seat
		seatIdx := RandomSeat(te.table.State.SeatMap)
		if seat != UnsetValue && te.table.State.SeatMap[seat] != UnsetValue {
			seatIdx = seat
		}
		te.table.State.SeatMap[seatIdx] = newPlayerIdx
		te.table.State.PlayerStates[newPlayerIdx].Seat = seatIdx
		te.table.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.CompetitionMeta.TableMaxSeatCount, te.table.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		te.table.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(te.table.State.PlayerStates[targetPlayerIdx].Seat, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.CompetitionMeta.TableMaxSeatCount, te.table.Meta.CompetitionMeta.Rule)
		te.table.State.PlayerStates[targetPlayerIdx].Bankroll += redeemChips
		te.table.State.PlayerStates[targetPlayerIdx].IsParticipated = true
	}

	te.emitEvent("PlayerReserve", joinPlayer.PlayerID)
	return nil
}

/*
	PlayersBatchReserve 多位玩家確認座位
	  - 適用時機: 拆併桌整桌玩家確認座位、開桌時有預設玩家
*/
func (te *tableEngine) PlayersBatchReserve(joinPlayers []JoinPlayer) error {
	if len(te.table.State.PlayerStates)+len(joinPlayers) > te.table.Meta.CompetitionMeta.TableMaxSeatCount {
		return ErrTableNoEmptySeats
	}

	copyTable := *te.table
	for _, joinPlayer := range joinPlayers {
		if err := te.PlayerReserve(joinPlayer); err != nil {
			te.table = &copyTable
			return err
		}
	}
	te.table.State.Status = TableStateStatus_TableBalancing

	te.emitEvent("PlayersBatchReserve", "")

	// Preparing ready group for waiting all players' join
	te.rg.Stop()
	te.rg.SetTimeoutInterval(10) // TODO: ask for longest period for timeout
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
		if te.table.State.Status == TableStateStatus_TableBalancing {
			if err := te.TableGameOpen(); err != nil {
				te.onTableErrorUpdated(te.table, err)
			}
		}
	})

	te.rg.ResetParticipants()
	for playerIdx := range te.table.State.PlayerStates {
		te.rg.Add(int64(playerIdx), false)
	}

	te.rg.Start()

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

	te.table.State.PlayerStates[playerIdx].IsIn = true

	if te.table.State.Status == TableStateStatus_TableBalancing {
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
	if te.table.State.PlayerStates[playerIdx].Bankroll == 0 {
		te.table.State.PlayerStates[playerIdx].IsBetweenDealerBB = IsBetweenDealerBB(te.table.State.PlayerStates[playerIdx].Seat, te.table.State.CurrentDealerSeat, te.table.State.CurrentBBSeat, te.table.Meta.CompetitionMeta.TableMaxSeatCount, te.table.Meta.CompetitionMeta.Rule)
	}
	te.table.State.PlayerStates[playerIdx].Bankroll += joinPlayer.RedeemChips

	te.emitEvent("PlayerRedeemChips", joinPlayer.PlayerID)
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
	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := te.table.FindPlayerIdx(playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return nil
	}

	// set leave PlayerIdx seatMap to UnsetValue
	leavePlayerIDMap := make(map[string]interface{})
	for _, leavePlayerIdx := range leavePlayerIndexes {
		leavePlayer := te.table.State.PlayerStates[leavePlayerIdx]
		leavePlayerIDMap[leavePlayer.PlayerID] = struct{}{}
		te.table.State.SeatMap[leavePlayer.Seat] = UnsetValue
	}

	// delete target players in PlayerStates
	te.table.State.PlayerStates = funk.Filter(te.table.State.PlayerStates, func(player *TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*TablePlayerState)

	// update current SeatMap player indexes in SeatMap
	for newPlayerIdx, player := range te.table.State.PlayerStates {
		te.table.State.SeatMap[player.Seat] = newPlayerIdx
	}

	te.table.State.Status = TableStateStatus_TableGameStandby
	te.emitEvent("PlayersLeave", strings.Join(playerIDs, ","))

	return nil
}

func (te *tableEngine) PlayerReady(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Ready(gamePlayerIdx)
}

func (te *tableEngine) PlayerPay(playerID string, chips int64) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Pay(gamePlayerIdx, chips)
}

func (te *tableEngine) PlayerBet(playerID string, chips int64) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Bet(gamePlayerIdx, chips)
}

func (te *tableEngine) PlayerRaise(playerID string, chipLevel int64) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Raise(gamePlayerIdx, chipLevel)
}

func (te *tableEngine) PlayerCall(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Call(gamePlayerIdx)
}

func (te *tableEngine) PlayerAllin(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Allin(gamePlayerIdx)
}

func (te *tableEngine) PlayerCheck(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Check(gamePlayerIdx)
}

func (te *tableEngine) PlayerFold(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Fold(gamePlayerIdx)
}

func (te *tableEngine) PlayerPass(playerID string) error {
	gamePlayerIdx := te.table.FindGamePlayerIdx(playerID)
	if err := te.validateGameMove(gamePlayerIdx); err != nil {
		return err
	}

	return te.game.Pass(gamePlayerIdx)
}

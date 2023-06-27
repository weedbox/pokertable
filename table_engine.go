package pokertable

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/syncsaga"
	"github.com/weedbox/timebank"
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

type TableEngine interface {
	// Others
	OnTableUpdated(fn func(*Table)) // 桌次更新事件監聽器
	OnErrorUpdated(fn func(error))  // 錯誤更新事件監聽器

	// Table Actions
	GetTable(tableID string) (*Table, error)               // 取得桌次
	CreateTable(tableSetting TableSetting) (*Table, error) // 建立桌
	BalanceTable(tableID string) error                     // 等待拆併桌中
	DeleteTable(tableID string) error                      // 刪除桌
	StartTableGame(tableID string) error                   // 開打遊戲
	TableGameOpen(tableID string) error                    // 開下一輪遊戲

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
	te := &tableEngine{
		timebank:       timebank.NewTimeBank(),
		onTableUpdated: func(*Table) {},
		onErrorUpdated: func(error) {},
		incoming:       make(chan *Request, 1024),
		tableGames:     sync.Map{},
	}
	go te.runRequestHandler()
	return te
}

type TableGame struct {
	Table          *Table
	Game           pokerface.Game
	GameReadyGroup *syncsaga.ReadyGroup
}

type tableEngine struct {
	lock           sync.Mutex
	timebank       *timebank.TimeBank
	onTableUpdated func(*Table)
	onErrorUpdated func(error)
	incoming       chan *Request
	tableGames     sync.Map
}

func (te *tableEngine) OnTableUpdated(fn func(*Table)) {
	te.onTableUpdated = fn
}

func (te *tableEngine) OnErrorUpdated(fn func(error)) {
	te.onErrorUpdated = fn
}

func (te *tableEngine) GetTable(tableID string) (*Table, error) {
	te.lock.Lock()
	defer te.lock.Unlock()

	tableGame, exist := te.tableGames.Load(tableID)
	if !exist {
		return nil, ErrTableNotFound
	}
	return tableGame.(*TableGame).Table, nil
}

func (te *tableEngine) CreateTable(tableSetting TableSetting) (*Table, error) {
	te.lock.Lock()
	defer te.lock.Unlock()

	// validate tableSetting
	if len(tableSetting.JoinPlayers) > tableSetting.CompetitionMeta.TableMaxSeatCount {
		return nil, ErrInvalidCreateTableSetting
	}

	// create table instance
	table := &Table{
		ID: uuid.New().String(),
	}
	table.ConfigureWithSetting(tableSetting, TableStateStatus_TableCreated)
	te.emitEvent("CreateTable", "", table)

	// create ready group
	rg := syncsaga.NewReadyGroup(
		syncsaga.WithTimeout(1, func(rg *syncsaga.ReadyGroup) {
			// Auto Ready By Default
			states := rg.GetParticipantStates()
			for gamePlayerIdx, isReady := range states {
				if !isReady {
					rg.Ready(gamePlayerIdx)
				}
			}
		}),
	)

	// create table game
	tableGame := TableGame{
		Table:          table,
		GameReadyGroup: rg,
	}

	// update tableGames
	te.tableGames.Store(table.ID, &tableGame)

	return table, nil
}

/*
	BalanceTable 等待拆併桌
	  - 適用時機: 該桌次需要拆併桌時
*/
func (te *tableEngine) BalanceTable(tableID string) error {
	return te.incomingRequest(tableID, RequestAction_BalanceTable, nil)
}

/*
	DeleteTable 刪除桌
	  - 適用時機: 強制關閉 (Killed)、自動關閉 (AutoEnded)、正常關閉 (Closed)
*/
func (te *tableEngine) DeleteTable(tableID string) error {
	return te.incomingRequest(tableID, RequestAction_DeleteTable, nil)
}

func (te *tableEngine) StartTableGame(tableID string) error {
	return te.incomingRequest(tableID, RequestAction_StartTableGame, nil)
}

func (te *tableEngine) TableGameOpen(tableID string) error {
	return te.incomingRequest(tableID, RequestAction_TableGameOpen, nil)
}

/*
	PlayerJoin 玩家入桌
	  - 適用時機: 報名入桌、補碼入桌
*/
func (te *tableEngine) PlayerJoin(tableID string, joinPlayer JoinPlayer) error {
	return te.incomingRequest(tableID, RequestAction_PlayerJoin, joinPlayer)
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機: 增購
*/
func (te *tableEngine) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	return te.incomingRequest(tableID, RequestAction_PlayerRedeemChips, joinPlayer)
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
	return te.incomingRequest(tableID, RequestAction_PlayersLeave, playerIDs)
}

func (te *tableEngine) PlayerReady(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerReady, playerID)
}

func (te *tableEngine) PlayerPay(tableID, playerID string, chips int64) error {
	param := PlayerPayParam{
		PlayerID: playerID,
		Chips:    chips,
	}
	return te.incomingRequest(tableID, RequestAction_PlayerPay, param)
}

func (te *tableEngine) PlayerBet(tableID, playerID string, chips int64) error {
	param := PlayerBetParam{
		PlayerID: playerID,
		Chips:    chips,
	}
	return te.incomingRequest(tableID, RequestAction_PlayerBet, param)
}

func (te *tableEngine) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	param := PlayerRaiseParam{
		PlayerID:  playerID,
		ChipLevel: chipLevel,
	}
	return te.incomingRequest(tableID, RequestAction_PlayerRaise, param)
}

func (te *tableEngine) PlayerCall(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerCall, playerID)
}

func (te *tableEngine) PlayerAllin(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerAllin, playerID)
}

func (te *tableEngine) PlayerCheck(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerCheck, playerID)
}

func (te *tableEngine) PlayerFold(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerFold, playerID)
}

func (te *tableEngine) PlayerPass(tableID, playerID string) error {
	return te.incomingRequest(tableID, RequestAction_PlayerPass, playerID)
}

func (te *tableEngine) incomingRequest(tableID string, action RequestAction, param interface{}) error {
	te.lock.Lock()
	defer te.lock.Unlock()

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
	fmt.Printf("->[Table %s][#%d][%d][%s] emit Event: %s\n", table.ID, table.UpdateSerial, table.State.GameCount, playerID, eventName)
	json, _ := table.GetGameStateJSON()
	fmt.Println(json)
	te.onTableUpdated(table)
}

func (te *tableEngine) emitErrorEvent(eventName RequestAction, playerID string, err error, table *Table) {
	table.RefreshUpdateAt()
	fmt.Printf("->[Table %s][#%d][%d][%s] emit ERROR Event: %s, Error: %v\n", table.ID, table.UpdateSerial, table.State.GameCount, playerID, eventName, err)
	te.onErrorUpdated(err)
}

func (te *tableEngine) runRequestHandler() {
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

package pokertable

import (
	"errors"

	"github.com/sirupsen/logrus"
)

var (
	ErrInvalidCreateTableSetting = errors.New("invalid create table setting")
	ErrPlayerNotFound            = errors.New("player not found")
	ErrNoEmptySeats              = errors.New("no empty seats available")
	ErrPlayerInvalidAction       = errors.New("player invalid action")
)

type TableEngine interface {
	// Table Actions
	OnTableUpdated(fn func(*Table)) error                  // 桌次更新監聽器
	CreateTable(tableSetting TableSetting) (Table, error)  // 建立桌
	CloseTable(table Table, status TableStateStatus) Table // 關閉桌
	StartGame(table Table) (Table, error)                  // 開打遊戲
	NextRound(table Table) (Table, error)                  // 遊戲下一階段
	TableSettlement(table Table) Table                     // 遊戲結算
	GameOpen(table Table) (Table, error)                   // 開下一輪遊戲

	// Player Actions
	// Player Table Actions
	PlayerJoin(table Table, joinPlayer JoinPlayer) (Table, error)        // 玩家入桌 (報名或補碼)
	PlayerRedeemChips(table Table, joinPlayer JoinPlayer) (Table, error) // 增購籌碼
	PlayersLeave(table Table, playerIDs []string) Table                  // 玩家們離桌

	// Player Game Actions
	PlayerReady(table Table, playerID string) (Table, error)                  // 玩家準備動作完成
	PlayerPay(table Table, playerID string, chips int64) (Table, error)       // 玩家付籌碼
	PlayerPayAnte(table Table, playerID string) (Table, error)                // 玩家付前注
	PlayerPaySB(table Table, playerID string) (Table, error)                  // 玩家付大盲
	PlayerPayBB(table Table, playerID string) (Table, error)                  // 玩家付小盲
	PlayerBet(table Table, playerID string, chips int64) (Table, error)       // 玩家下注
	PlayerRaise(table Table, playerID string, chipLevel int64) (Table, error) // 玩家加注
	PlayerCall(table Table, playerID string) (Table, error)                   // 玩家跟注
	PlayerAllin(table Table, playerID string) (Table, error)                  // 玩家全下
	PlayerCheck(table Table, playerID string) (Table, error)                  // 玩家過牌
	PlayerFold(table Table, playerID string) (Table, error)                   // 玩家棄牌

}

func NewTableEngine(gameEngine GameEngine, logLevel uint32) TableEngine {
	logger := logrus.New()
	logger.SetLevel(logrus.Level(logLevel))
	return &tableEngine{
		logger:     logger,
		gameEngine: gameEngine,
	}
}

type tableEngine struct {
	logger         *logrus.Logger
	gameEngine     GameEngine
	onTableUpdated func(*Table)
}

func (te *tableEngine) OnTableUpdated(fn func(*Table)) error {
	te.onTableUpdated = fn
	return nil
}

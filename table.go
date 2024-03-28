package pokertable

import (
	"encoding/json"
	"fmt"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

type TableStateStatus string

const (
	// TableStateStatus: Table
	TableStateStatus_TableCreated   TableStateStatus = "table_created"   // 桌次已建立
	TableStateStatus_TablePausing   TableStateStatus = "table_pausing"   // 桌次暫停中
	TableStateStatus_TableRestoring TableStateStatus = "table_restoring" // 桌次轉移中 (Graceful Shutdown)
	TableStateStatus_TableBalancing TableStateStatus = "table_balancing" // 桌次拆併桌中
	TableStateStatus_TableClosed    TableStateStatus = "table_closed"    // 桌次已結束

	// TableStateStatus: Game
	TableStateStatus_TableGameOpened  TableStateStatus = "table_game_opened"  // 桌次內遊戲已開局
	TableStateStatus_TableGamePlaying TableStateStatus = "table_game_playing" // 桌次內遊戲開打中
	TableStateStatus_TableGameSettled TableStateStatus = "table_game_settled" // 桌次內遊戲已結算
	TableStateStatus_TableGameStandby TableStateStatus = "table_game_standby" // 桌次內遊戲準備中
)

type Table struct {
	UpdateSerial int64       `json:"update_serial"` // 更新序列號 (數字越大越晚發生)
	ID           string      `json:"id"`            // 桌次 Unique ID
	Meta         TableMeta   `json:"meta"`          // 桌次固定資料
	State        *TableState `json:"state"`         // 桌次動態資料
	UpdateAt     int64       `json:"update_at"`     // 更新時間 (Seconds)
}

type TableMeta struct {
	CompetitionID       string `json:"competition_id"`         // 賽事 ID
	Rule                string `json:"rule"`                   // 德州撲克規則, 常牌(default), 短牌(short_deck), 奧瑪哈(omaha)
	Mode                string `json:"mode"`                   // 賽事模式 (CT, MTT, Cash)
	MaxDuration         int    `json:"max_duration"`           // 比賽時間總長 (Seconds)
	TableMaxSeatCount   int    `json:"table_max_seat_count"`   // 每桌人數上限
	TableMinPlayerCount int    `json:"table_min_player_count"` // 每桌最小開打數
	MinChipUnit         int64  `json:"min_chip_unit"`          // 最小單位籌碼量
	ActionTime          int    `json:"action_time"`            // 玩家動作思考時間 (Seconds)
}

type TableState struct {
	Status               TableStateStatus       `json:"status"`                   // 當前桌次狀態
	StartAt              int64                  `json:"start_at"`                 // 開打時間 (Seconds)
	SeatMap              []int                  `json:"seat_map"`                 // 座位入座狀況，index: seat index (0-8), value: TablePlayerState index (-1 by default)
	BlindState           *TableBlindState       `json:"blind_state"`              // 盲注狀態
	CurrentDealerSeat    int                    `json:"current_dealer_seat"`      // 當前 Dealer 座位編號
	CurrentBBSeat        int                    `json:"current_bb_seat"`          // 當前 BB 座位編號
	PlayerStates         []*TablePlayerState    `json:"player_states"`            // 賽局桌上玩家狀態
	GameCount            int                    `json:"game_count"`               // 執行牌局遊戲次數 (遊戲跑幾輪)
	GamePlayerIndexes    []int                  `json:"game_player_indexes"`      // 本手正在玩的 PlayerIndex 陣列 (陣列 index 為從 Dealer 位置開始的 PlayerIndex)，GameEngine 用
	GameState            *pokerface.GameState   `json:"game_state"`               // 本手狀態
	LastPlayerGameAction *TablePlayerGameAction `json:"last_player_game_action"`  // 最新一筆玩家牌局動作
	NextBBOrderPlayerIDs []string               `json:"next_bb_order_player_ids"` // 下一手 BB 座位玩家 ID 陣列
	// GameStatistics       *TableGameStatistics   `json:"game_statistics"`          // 每手遊戲統計
}

type TablePlayerGameAction struct {
	CompetitionID    string   `json:"competition_id"`     // 賽事 ID
	TableID          string   `json:"table_id"`           // 桌次 ID
	GameID           string   `json:"game_id"`            // 遊戲 ID
	GameCount        int      `json:"game_count"`         // 執行牌局遊戲次數 (遊戲跑幾輪)
	Round            string   `json:"round"`              // 哪回合
	UpdateAt         int64    `json:"update_at"`          // 更新時間 (Seconds)
	PlayerID         string   `json:"player_id"`          // 玩家 ID
	Seat             int      `json:"seat"`               // 座位編號 0 ~ 8
	Positions        []string `json:"positions"`          // 場上位置
	Action           string   `json:"action"`             // 動作
	Chips            int64    `json:"chips"`              // 下注籌碼量
	Bankroll         int64    `json:"bankroll"`           // GameState.Player Bankroll
	InitialStackSize int64    `json:"initial_stack_size"` // GameState.Player InitialStackSize
	StackSize        int64    `json:"stack_size"`         // GameState.Player StackSize
	Pot              int64    `json:"pot"`                // GameState.Player Pot
	Wager            int64    `json:"wager"`              // GameState.Player Wager
}

type TablePlayerState struct {
	PlayerID          string                    `json:"player_id"`            // 玩家 ID
	Seat              int                       `json:"seat"`                 // 座位編號 0 ~ 8
	Positions         []string                  `json:"positions"`            // 場上位置
	IsParticipated    bool                      `json:"is_participated"`      // 玩家是否參戰
	IsBetweenDealerBB bool                      `json:"is_between_dealer_bb"` // 玩家入場時是否在 Dealer & BB 之間
	Bankroll          int64                     `json:"bankroll"`             // 玩家身上籌碼
	IsIn              bool                      `json:"is_in"`                // 玩家是否入座
	GameStatistics    TablePlayerGameStatistics `json:"game_statistics"`      // 玩家每手遊戲統計
}

type TableBlindState struct {
	Level  int   `json:"level"`  // 盲注等級(-1 表示中場休息)
	Ante   int64 `json:"ante"`   // 前注籌碼量
	Dealer int64 `json:"dealer"` // 庄位籌碼量
	SB     int64 `json:"sb"`     // 大盲籌碼量
	BB     int64 `json:"bb"`     // 小盲籌碼量
}

// Table Getters
func (t Table) Clone() (*Table, error) {
	encoded, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	var cloneTable Table
	if err := json.Unmarshal(encoded, &cloneTable); err != nil {
		return nil, err
	}

	return &cloneTable, nil
}

func (t Table) GetJSON() (string, error) {
	encoded, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (t Table) GetGameStateJSON() (string, error) {
	encoded, err := json.Marshal(t.State.GameState)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (t Table) ParticipatedPlayers() []*TablePlayerState {
	return funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
		return player.IsParticipated
	}).([]*TablePlayerState)
}

func (t Table) AlivePlayers() []*TablePlayerState {
	return funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
		return player.Bankroll > 0
	}).([]*TablePlayerState)
}

func (t Table) GamePlayerIndex(playerID string) int {
	targetPlayerIdx := UnsetValue
	for idx, player := range t.State.PlayerStates {
		if player.PlayerID == playerID {
			targetPlayerIdx = idx
			break
		}
	}

	if targetPlayerIdx == UnsetValue {
		return UnsetValue
	}

	for gamePlayerIndex, playerIndex := range t.State.GamePlayerIndexes {
		if targetPlayerIdx == playerIndex {
			return gamePlayerIndex
		}
	}
	return UnsetValue
}

func (t Table) FindGamePlayerIdx(playerID string) int {
	for gamePlayerIdx, playerIdx := range t.State.GamePlayerIndexes {
		if playerIdx >= len(t.State.PlayerStates) {
			fmt.Printf("[DEBUG#FindGamePlayerIdx] TableID: %s, PlayerID: %s, GamePlayerIndexes: %+v, len(PlayerStates): %d, TableSerial: %d\n",
				t.ID,
				playerID,
				t.State.GamePlayerIndexes,
				len(t.State.PlayerStates),
				t.UpdateSerial,
			)
			continue
		}
		player := t.State.PlayerStates[playerIdx]
		if player.PlayerID == playerID {
			return gamePlayerIdx
		}
	}
	return UnsetValue
}

func (t Table) FindPlayerIdx(playerID string) int {
	for idx, player := range t.State.PlayerStates {
		if player.PlayerID == playerID {
			return idx
		}
	}
	return UnsetValue
}

func (t Table) PlayerSeatMap() map[string]int {
	playerSeatMap := make(map[string]int)
	for _, player := range t.State.PlayerStates {
		playerSeatMap[player.PlayerID] = player.Seat
	}
	return playerSeatMap
}

func (t Table) FindPlayerIndexFromGamePlayerIndex(gamePlayerIdx int) int {
	// game player index is out of range
	if gamePlayerIdx < 0 || gamePlayerIdx >= len(t.State.PlayerStates) {
		return UnsetValue
	}

	playerIdx := t.State.GamePlayerIndexes[gamePlayerIdx]

	// player index is out of range
	if playerIdx >= len(t.State.PlayerStates) {
		return UnsetValue
	}

	return playerIdx
}

/*
ShouldPause 計算本桌是否已達到暫停
  - 暫停條件 1: 中場休息
  - 暫停條件 2: 存活玩家小於最小開打數
*/
func (t Table) ShouldPause() bool {
	return t.State.BlindState.IsBreaking() || len(t.AlivePlayers()) < t.Meta.TableMinPlayerCount
}

// TableBlindState Getters
func (bs TableBlindState) IsBreaking() bool {
	return bs.Level == -1
}

func (bs TableBlindState) IsSet() bool {
	return bs.Level != 0 && bs.Ante != UnsetValue && bs.Dealer != UnsetValue && bs.SB != UnsetValue && bs.BB != UnsetValue
}

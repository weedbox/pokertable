package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/weedbox/pokertable/util"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

type TableStateStatus string

const (
	// TableStateStatus
	TableStateStatus_TableGameCreated                 TableStateStatus = "TableGame_Created"                 // 本桌遊戲已建立
	TableStateStatus_TableGameKilled                  TableStateStatus = "TableGame_Killed"                  // 本桌遊戲已被強制關閉
	TableStateStatus_TableGameAutoEnded               TableStateStatus = "TableGame_AutoEnded"               // 本桌遊戲已被自動關閉
	TableStateStatus_TableGamePaused                  TableStateStatus = "TableGame_Paused"                  // 本桌遊戲暫停
	TableStateStatus_TableGameRestoring               TableStateStatus = "TableGame_Restoring"               // 本桌遊戲轉移中 (Graceful Shutdown)
	TableStateStatus_TableGameClosed                  TableStateStatus = "TableGame_Closed"                  // 本桌遊戲結束
	TableStateStatus_TableGameMatchOpen               TableStateStatus = "TableGame_MatchOpen"               // 本手遊戲已開始
	TableStateStatus_TableGameWaitingDistributeTables TableStateStatus = "TableGame_WaitingDistributeTables" // 等待拆併桌
)

type Table struct {
	ID       string      `json:"id"`
	Meta     TableMeta   `json:"meta"`
	State    *TableState `json:"state"`
	UpdateAt int64       `json:"update_at"`
}

type TableMeta struct {
	ShortID         string          `json:"short_id"`         // Table ID 簡短版 (大小寫英文或數字組成6位數)
	Code            string          `json:"code"`             // 桌次編號
	Name            string          `json:"name"`             // 桌次名稱
	InvitationCode  string          `json:"invitation_code"`  // 桌次邀請碼
	CompetitionMeta CompetitionMeta `json:"competition_meta"` // 賽事固定資料
}

type CompetitionMeta struct {
	Rule                 string `json:"rule"`                    // 德州撲克規則, 常牌(default), 短牌(short_deck), 奧瑪哈(omaha)
	Mode                 string `json:"mode"`                    // 賽事模式 (CT, MTT, Cash)
	MaxDurationMins      int    `json:"max_duration_mins"`       // 比賽時間總長 (分鐘)
	TableMaxSeatCount    int    `json:"table_max_seat_count"`    // 每桌人數上限
	TableMinPlayingCount int    `json:"table_min_playing_count"` // 每桌最小開打數
	MinChipsUnit         int64  `json:"min_chips_unit"`          // 最小單位籌碼量
	Blind                Blind  `json:"blind"`                   // 盲注資訊
}

type Blind struct {
	ID               string       `json:"id"`                 // ID
	Name             string       `json:"name"`               // 名稱
	FinalBuyInLevel  int          `json:"final_buy_in_level"` // 最後買入盲注等級
	DealerBlindTimes int          `json:"dealer_blind_times"` // Dealer 位置要收取的前注倍數 (短牌用)
	Levels           []BlindLevel `json:"levels"`             // 級別資訊列表
}

type BlindLevel struct {
	Level        int   `json:"level"`         // 盲注等級(-1 表示中場休息)
	SBChips      int64 `json:"sb_chips"`      // 小盲籌碼量
	BBChips      int64 `json:"bb_chips"`      // 大盲籌碼量
	AnteChips    int64 `json:"ante_chips"`    // 前注籌碼量
	DurationMins int   `json:"duration_mins"` // 等級持續時間
}

type TableState struct {
	GameCount              int                 `json:"game_count"`                // 執行牌局遊戲次數 (遊戲跑幾輪)
	StartGameAt            int64               `json:"start_game_at"`             // 開打時間
	BlindState             *TableBlindState    `json:"blind_state"`               // 盲注狀態
	CurrentDealerSeatIndex int                 `json:"current_dealer_seat_index"` // 當前 Dealer 座位編號
	CurrentBBSeatIndex     int                 `json:"current_bb_seat_index"`     // 當前 BB 座位編號
	PlayerSeatMap          []int               `json:"player_seat_map"`           // 座位入座狀況，index: seat index (0-8), value: TablePlayerState index (-1 by default)
	PlayerStates           []*TablePlayerState `json:"player_states"`             // 賽局桌上玩家狀態
	PlayingPlayerIndexes   []int               `json:"playing_player_indexes"`    // 本手正在玩的 PlayerIndex 陣列 (陣列 index 為從 Dealer 位置開始的 PlayerIndex)，GameEngine 用
	Status                 TableStateStatus    `json:"status"`                    // 當前桌次狀態
	Rankings               []int               `json:"ranks"`                     // 當桌即時排名, 名次: 陣列 index + 1, 陣列元素: player_idx, ex: [4, 0, 2]: 第一名: players[4], 第二名: players[0]...
	GameState              pokerface.GameState `json:"game_state"`                // 本手狀態 (pokerface.GameState)
}

type TablePlayerState struct {
	PlayerID          string   `json:"player_id"`            // 玩家 ID
	SeatIndex         int      `json:"seat_index"`           // 座位編號 0 ~ 8
	Positions         []string `json:"positions"`            // 場上位置
	IsParticipated    bool     `json:"is_participated"`      // 玩家是否參戰，入座 ≠ 參戰
	IsBetweenDealerBB bool     `json:"is_between_dealer_bb"` // 玩家入場時是否在 Dealer & BB 之間
	Bankroll          int64    `json:"bankroll"`             // 玩家身上籌碼
}

type TableBlindState struct {
	FinalBuyInLevelIndex int                     `json:"final_buy_in_level_idx"` // 最後買入盲注等級索引值
	InitialLevel         int                     `json:"initial_level"`          // 起始盲注級別
	CurrentLevelIndex    int                     `json:"current_level_index"`    // 現在盲注等級級別索引值
	LevelStates          []*TableBlindLevelState `json:"level_states"`           // 級別資訊列表狀態
}

type TableBlindLevelState struct {
	Level        int   `json:"level"`         // 盲注等級(-1 表示中場休息)
	SBChips      int64 `json:"sb_chips"`      // 小盲籌碼量
	BBChips      int64 `json:"bb_chips"`      // 大盲籌碼量
	AnteChips    int64 `json:"ante_chips"`    // 前注籌碼量
	DurationMins int   `json:"duration_mins"` // 等級持續時間
	LevelEndAt   int64 `json:"level_end_at"`  // 等級結束時間
}

// Setters
func (table *Table) Update() {
	table.UpdateAt = time.Now().Unix()
}

func (table *Table) Reset() {
	table.State.PlayingPlayerIndexes = []int{}
	for i := 0; i < len(table.State.PlayerStates); i++ {
		table.State.PlayerStates[i].Positions = make([]string, 0)
	}
}

// Table Getters
func (table Table) ModeRule() string {
	return fmt.Sprintf("%s_%s_holdem", table.Meta.CompetitionMeta.Mode, table.Meta.CompetitionMeta.Rule)
}

func (table Table) GetJSON() (*string, error) {
	encoded, err := json.Marshal(table)
	if err != nil {
		return nil, err
	}
	json := string(encoded)
	return &json, nil
}

func (table Table) ParticipatedPlayers() []*TablePlayerState {
	return funk.Filter(table.State.PlayerStates, func(player *TablePlayerState) bool {
		return player.IsParticipated
	}).([]*TablePlayerState)
}

func (t Table) EndGameAt() int64 {
	return time.Unix(t.State.StartGameAt, 0).Add(time.Minute * time.Duration(t.Meta.CompetitionMeta.MaxDurationMins)).Unix()
}

func (t Table) AlivePlayers() []*TablePlayerState {
	return funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
		return player.Bankroll > 0
	}).([]*TablePlayerState)
}

// TableBlindState Setters
func (bs *TableBlindState) Update() {
	// 更新現在盲注資訊
	now := time.Now().Unix()
	for idx, levelState := range bs.LevelStates {
		timeDiff := now - levelState.LevelEndAt
		if timeDiff < 0 {
			bs.CurrentLevelIndex = idx
			break
		} else {
			bs.CurrentLevelIndex = idx + 1
		}
	}
}

// TableBlindState Getters
func (bs TableBlindState) IsFinalBuyInLevel() bool {
	if bs.FinalBuyInLevelIndex == util.UnsetValue {
		return false
	}

	return bs.CurrentLevelIndex > bs.FinalBuyInLevelIndex
}

func (bs TableBlindState) IsBreaking() bool {
	return bs.LevelStates[bs.CurrentLevelIndex].Level == -1
}

func (bs TableBlindState) CurrentBlindLevel() TableBlindLevelState {
	return *bs.LevelStates[bs.CurrentLevelIndex]
}

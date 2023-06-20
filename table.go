package pokertable

import (
	"encoding/json"
	"time"

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
	TableStateStatus_TableGameStandby TableStateStatus = "table_game_standby" // 桌次內遊戲尚未開始
	TableStateStatus_TableGameOpened  TableStateStatus = "table_game_opened"  // 桌次內遊戲已開局
	TableStateStatus_TableGamePlaying TableStateStatus = "table_game_playing" // 桌次內遊戲開打中
	TableStateStatus_TableGameSettled TableStateStatus = "table_game_settled" // 桌次內遊戲已結算
)

type Table struct {
	ID           string      `json:"id"`
	Meta         TableMeta   `json:"meta"`
	State        *TableState `json:"state"`
	UpdateAt     int64       `json:"update_at"`     // 更新時間 (Seconds)
	UpdateSerial int64       `json:"update_serial"` // 更新序列號 (數字越大越晚發生)
}

type TableMeta struct {
	ShortID         string          `json:"short_id"`         // Table ID 簡短版 (大小寫英文或數字組成6位數)
	Code            string          `json:"code"`             // 桌次編號
	Name            string          `json:"name"`             // 桌次名稱
	InvitationCode  string          `json:"invitation_code"`  // 桌次邀請碼
	CompetitionMeta CompetitionMeta `json:"competition_meta"` // 賽事固定資料
}

type CompetitionMeta struct {
	ID                  string `json:"id"`                     // 賽事 ID
	Rule                string `json:"rule"`                   // 德州撲克規則, 常牌(default), 短牌(short_deck), 奧瑪哈(omaha)
	Mode                string `json:"mode"`                   // 賽事模式 (CT, MTT, Cash)
	MaxDuration         int    `json:"max_duration"`           // 比賽時間總長 (Seconds)
	TableMaxSeatCount   int    `json:"table_max_seat_count"`   // 每桌人數上限
	TableMinPlayerCount int    `json:"table_min_player_count"` // 每桌最小開打數
	MinChipUnit         int64  `json:"min_chip_unit"`          // 最小單位籌碼量
	Blind               Blind  `json:"blind"`                  // 盲注資訊
	ActionTime          int    `json:"action_time"`            // 玩家動作思考時間 (Seconds)
}

type Blind struct {
	ID              string       `json:"id"`                 // ID
	Name            string       `json:"name"`               // 名稱
	InitialLevel    int          `json:"initial_level"`      // 起始盲注級別
	FinalBuyInLevel int          `json:"final_buy_in_level"` // 最後買入盲注等級
	DealerBlindTime int          `json:"dealer_blind_time"`  // Dealer 位置要收取的前注倍數 (短牌用)
	Levels          []BlindLevel `json:"levels"`             // 級別資訊列表
}

type BlindLevel struct {
	Level    int   `json:"level"`    // 盲注等級(-1 表示中場休息)
	SB       int64 `json:"sb"`       // 小盲籌碼量
	BB       int64 `json:"bb"`       // 大盲籌碼量
	Ante     int64 `json:"ante"`     // 前注籌碼量
	Duration int   `json:"duration"` // 等級持續時間 (Seconds)
}

type TableState struct {
	Status            TableStateStatus     `json:"status"`              // 當前桌次狀態
	StartAt           int64                `json:"start_at"`            // 開打時間 (Seconds)
	SeatMap           []int                `json:"seat_map"`            // 座位入座狀況，index: seat index (0-8), value: TablePlayerState index (-1 by default)
	BlindState        *TableBlindState     `json:"blind_state"`         // 盲注狀態
	CurrentDealerSeat int                  `json:"current_dealer_seat"` // 當前 Dealer 座位編號
	CurrentBBSeat     int                  `json:"current_bb_seat"`     // 當前 BB 座位編號
	PlayerStates      []*TablePlayerState  `json:"player_states"`       // 賽局桌上玩家狀態
	GameCount         int                  `json:"game_count"`          // 執行牌局遊戲次數 (遊戲跑幾輪)
	GamePlayerIndexes []int                `json:"game_player_indexes"` // 本手正在玩的 PlayerIndex 陣列 (陣列 index 為從 Dealer 位置開始的 PlayerIndex)，GameEngine 用
	GameState         *pokerface.GameState `json:"game_state"`          // 本手狀態
}

type TablePlayerState struct {
	PlayerID          string   `json:"player_id"`            // 玩家 ID
	Seat              int      `json:"seat"`                 // 座位編號 0 ~ 8
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
	BlindLevel BlindLevel `json:"blind_level"` // 盲注等級資訊
	EndAt      int64      `json:"end_at"`      // 等級結束時間 (Seconds)
}

// Setters
func (t *Table) RefreshUpdateAt() {
	t.UpdateAt = time.Now().Unix()
	t.UpdateSerial++
}

func (t *Table) Reset() {
	t.State.GamePlayerIndexes = []int{}
	for i := 0; i < len(t.State.PlayerStates); i++ {
		t.State.PlayerStates[i].Positions = make([]string, 0)
	}
}

func (t *Table) ConfigureWithSetting(setting TableSetting, status TableStateStatus) {
	// configure meta
	meta := TableMeta{
		ShortID:         setting.ShortID,
		Code:            setting.Code,
		Name:            setting.Name,
		InvitationCode:  setting.InvitationCode,
		CompetitionMeta: setting.CompetitionMeta,
	}
	t.Meta = meta

	// configure state
	finalBuyInLevelIdx := UnsetValue
	if setting.CompetitionMeta.Blind.FinalBuyInLevel != UnsetValue {
		for idx, blindLevel := range setting.CompetitionMeta.Blind.Levels {
			if blindLevel.Level == setting.CompetitionMeta.Blind.FinalBuyInLevel {
				finalBuyInLevelIdx = idx
				break
			}
		}
	}

	blindState := TableBlindState{
		FinalBuyInLevelIndex: finalBuyInLevelIdx,
		InitialLevel:         setting.CompetitionMeta.Blind.InitialLevel,
		CurrentLevelIndex:    UnsetValue,
		LevelStates: funk.Map(setting.CompetitionMeta.Blind.Levels, func(blindLevel BlindLevel) *TableBlindLevelState {
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
		SeatMap:           NewDefaultSeatMap(setting.CompetitionMeta.TableMaxSeatCount),
		PlayerStates:      make([]*TablePlayerState, 0),
		GamePlayerIndexes: make([]int, 0),
		Status:            status,
	}
	t.State = &state

	// handle auto join players
	if len(setting.JoinPlayers) > 0 {
		// auto join players
		t.State.PlayerStates = funk.Map(setting.JoinPlayers, func(p JoinPlayer) *TablePlayerState {
			return &TablePlayerState{
				PlayerID:          p.PlayerID,
				Seat:              UnsetValue,
				Positions:         []string{Position_Unknown},
				IsParticipated:    true,
				IsBetweenDealerBB: false,
				Bankroll:          p.RedeemChips,
			}
		}).([]*TablePlayerState)

		// update seats
		for playerIdx := 0; playerIdx < len(t.State.PlayerStates); playerIdx++ {
			seatIdx := RandomSeat(state.SeatMap)
			t.State.SeatMap[seatIdx] = playerIdx
			t.State.PlayerStates[playerIdx].Seat = seatIdx
		}
	}
}

func (t *Table) ActivateBlindState() {
	for idx, levelState := range t.State.BlindState.LevelStates {
		if levelState.BlindLevel.Level == t.State.BlindState.InitialLevel {
			t.State.BlindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := t.State.StartAt
	for i := (t.State.BlindState.InitialLevel - 1); i < len(t.State.BlindState.LevelStates); i++ {
		if i == t.State.BlindState.InitialLevel-1 {
			t.State.BlindState.LevelStates[i].EndAt = blindStartAt
		} else {
			t.State.BlindState.LevelStates[i].EndAt = t.State.BlindState.LevelStates[i-1].EndAt
		}
		blindPassedSeconds := int64(t.State.BlindState.LevelStates[i].BlindLevel.Duration)
		t.State.BlindState.LevelStates[i].EndAt += blindPassedSeconds
	}
}

func (t *Table) OpenGame() {
	// Step 1: 更新狀態
	t.State.Status = TableStateStatus_TableGameOpened

	// Step 2: 檢查參賽資格
	for i := 0; i < len(t.State.PlayerStates); i++ {
		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if t.State.PlayerStates[i].IsParticipated || t.State.PlayerStates[i].IsBetweenDealerBB {
			t.State.PlayerStates[i].IsParticipated = t.State.PlayerStates[i].Bankroll > 0
			continue
		}

		// 檢查後手 (有錢的玩家可參賽)
		t.State.PlayerStates[i].IsParticipated = t.State.PlayerStates[i].Bankroll > 0
	}

	// Step 3: 處理可參賽玩家剩餘一人時，桌上有其他玩家情形
	if len(t.ParticipatedPlayers()) < 2 {
		for i := 0; i < len(t.State.PlayerStates); i++ {
			if t.State.PlayerStates[i].Bankroll == 0 {
				continue
			}

			t.State.PlayerStates[i].IsParticipated = true
			t.State.PlayerStates[i].IsBetweenDealerBB = false
		}
	}

	// Step 4: 計算新 Dealer Seat & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(t.State.GameCount, t.State.CurrentDealerSeat, t.Meta.CompetitionMeta.TableMinPlayerCount, t.Meta.CompetitionMeta.TableMaxSeatCount, t.State.PlayerStates, t.State.SeatMap)
	newDealerTableSeatIdx := t.State.PlayerStates[newDealerPlayerIdx].Seat

	// Step 5: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(t.State.PlayerStates); i++ {
		if !t.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if newDealerTableSeatIdx-t.State.CurrentDealerSeat < 0 {
			for j := t.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx+t.Meta.CompetitionMeta.TableMaxSeatCount; j++ {
				if (j % t.Meta.CompetitionMeta.TableMaxSeatCount) != t.State.PlayerStates[i].Seat {
					continue
				}

				t.State.PlayerStates[i].IsParticipated = true
				t.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		} else {
			for j := t.State.CurrentDealerSeat + 1; j < newDealerTableSeatIdx; j++ {
				if j != t.State.PlayerStates[i].Seat {
					continue
				}

				t.State.PlayerStates[i].IsParticipated = true
				t.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		}
	}

	// Step 6: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	gamePlayerIndexes := FindGamePlayerIndexes(newDealerTableSeatIdx, t.State.SeatMap, t.State.PlayerStates)
	t.State.GamePlayerIndexes = FindGamePlayerIndexes(newDealerTableSeatIdx, t.State.SeatMap, t.State.PlayerStates)

	// Step 7: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(t.Meta.CompetitionMeta.Rule, t.State.PlayerStates, gamePlayerIndexes)
	for playerIdx := 0; playerIdx < len(t.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && t.State.PlayerStates[playerIdx].IsParticipated {
			t.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 8: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置)
	t.State.GameCount = t.State.GameCount + 1
	t.State.CurrentDealerSeat = newDealerTableSeatIdx
	if len(gamePlayerIndexes) == 2 {
		bbPlayerIdx := gamePlayerIndexes[1]
		t.State.CurrentBBSeat = t.State.PlayerStates[bbPlayerIdx].Seat
	} else if len(gamePlayerIndexes) > 2 {
		bbPlayerIdx := gamePlayerIndexes[2]
		t.State.CurrentBBSeat = t.State.PlayerStates[bbPlayerIdx].Seat
	} else {
		t.State.CurrentBBSeat = UnsetValue
	}
}

func (t *Table) SettleGameResult() {
	t.State.Status = TableStateStatus_TableGameSettled

	// Step 1: 更新盲注 Level
	t.State.BlindState.Update()

	// Step 2: 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range t.State.GameState.Result.Players {
		playerIdx := t.State.GamePlayerIndexes[player.Idx]
		t.State.PlayerStates[playerIdx].Bankroll = player.Final
	}
}

func (t *Table) ContinueGame() {
	shouldPause := t.State.BlindState.IsBreaking() || (!t.State.BlindState.IsFinalBuyInLevel() && len(t.AlivePlayers()) < 2)
	if shouldPause {
		t.State.Status = TableStateStatus_TablePausing
	} else {
		t.State.Status = TableStateStatus_TableGameStandby
	}

	t.Reset()
}

func (t *Table) PlayerJoin(playerID string, redeemChips int64) error {
	// find player index in PlayerStates
	targetPlayerIdx := t.findPlayerIdx(playerID)

	if targetPlayerIdx == UnsetValue {
		if len(t.State.PlayerStates) == t.Meta.CompetitionMeta.TableMaxSeatCount {
			return ErrNoEmptySeats
		}

		// BuyIn
		player := TablePlayerState{
			PlayerID:          playerID,
			Seat:              UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          redeemChips,
		}
		t.State.PlayerStates = append(t.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(t.State.PlayerStates) - 1
		seatIdx := RandomSeat(t.State.SeatMap)
		t.State.SeatMap[seatIdx] = newPlayerIdx
		t.State.PlayerStates[newPlayerIdx].Seat = seatIdx
		t.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, t.State.CurrentDealerSeat, t.State.CurrentBBSeat, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		t.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(t.State.PlayerStates[targetPlayerIdx].Seat, t.State.CurrentDealerSeat, t.State.CurrentBBSeat, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
		t.State.PlayerStates[targetPlayerIdx].Bankroll += redeemChips
		t.State.PlayerStates[targetPlayerIdx].IsParticipated = true
	}

	return nil
}

func (t *Table) PlayerRedeemChips(playerIdx int, redeemChips int64) {
	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	if t.State.PlayerStates[playerIdx].Bankroll == 0 {
		t.State.PlayerStates[playerIdx].IsBetweenDealerBB = IsBetweenDealerBB(t.State.PlayerStates[playerIdx].Seat, t.State.CurrentDealerSeat, t.State.CurrentBBSeat, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
	}
	t.State.PlayerStates[playerIdx].Bankroll += redeemChips
}

func (t *Table) PlayersLeave(leavePlayerIndexes []int) {
	// set leave PlayerIdx int seatMap to UnsetValue
	leavePlayerIDMap := make(map[string]interface{})
	for _, leavePlayerIdx := range leavePlayerIndexes {
		leavePlayer := t.State.PlayerStates[leavePlayerIdx]
		leavePlayerIDMap[leavePlayer.PlayerID] = struct{}{}
		t.State.SeatMap[leavePlayer.Seat] = UnsetValue
	}

	// delete target players in PlayerStates
	t.State.PlayerStates = funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*TablePlayerState)

	// update current SeatMap player indexes in SeatMap
	for newPlayerIdx, player := range t.State.PlayerStates {
		t.State.SeatMap[player.Seat] = newPlayerIdx
	}
}

// Table Getters
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

func (t Table) EndGameAt() int64 {
	return time.Unix(t.State.StartAt, 0).Add(time.Second * time.Duration(t.Meta.CompetitionMeta.MaxDuration)).Unix()
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

/*
	IsClose 計算本桌是否已達到結束條件
	  - 結束條件 1: 達到結束時間
	  - 結束條件 2: 停止買入後且存活玩家剩餘 1 人
*/
func (t Table) IsClose() bool {
	return time.Now().Unix() > t.EndGameAt() || (t.State.BlindState.IsFinalBuyInLevel() && len(t.AlivePlayers()) == 1)
}

func (t Table) findPlayerIdx(playerID string) int {
	for idx, player := range t.State.PlayerStates {
		if player.PlayerID == playerID {
			return idx
		}
	}
	return UnsetValue
}

// TableBlindState Setters
func (bs *TableBlindState) Update() {
	// 更新現在盲注資訊
	now := time.Now().Unix()
	for idx, levelState := range bs.LevelStates {
		timeDiff := now - levelState.EndAt
		if timeDiff < 0 {
			bs.CurrentLevelIndex = idx
			break
		} else {
			if idx+1 < len(bs.LevelStates) {
				bs.CurrentLevelIndex = idx + 1
			}
		}
	}
}

// TableBlindState Getters
func (bs TableBlindState) IsFinalBuyInLevel() bool {
	// 沒有預設 FinalBuyInLevelIndex 代表不能補碼，永遠都是停止買入階段
	if bs.FinalBuyInLevelIndex == UnsetValue {
		return true
	}

	return bs.CurrentLevelIndex > bs.FinalBuyInLevelIndex
}

func (bs TableBlindState) IsBreaking() bool {
	return bs.LevelStates[bs.CurrentLevelIndex].BlindLevel.Level == -1
}

func (bs TableBlindState) CurrentBlindLevel() TableBlindLevelState {
	return *bs.LevelStates[bs.CurrentLevelIndex]
}

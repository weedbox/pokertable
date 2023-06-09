package pokertable

import (
	"encoding/json"
	"fmt"
	"time"

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
	ID                   string `json:"id"`                      // 賽事 ID
	Rule                 string `json:"rule"`                    // 德州撲克規則, 常牌(default), 短牌(short_deck), 奧瑪哈(omaha)
	Mode                 string `json:"mode"`                    // 賽事模式 (CT, MTT, Cash)
	MaxDurationMins      int    `json:"max_duration_mins"`       // 比賽時間總長 (分鐘)
	TableMaxSeatCount    int    `json:"table_max_seat_count"`    // 每桌人數上限
	TableMinPlayingCount int    `json:"table_min_playing_count"` // 每桌最小開打數
	MinChipsUnit         int64  `json:"min_chips_unit"`          // 最小單位籌碼量
	Blind                Blind  `json:"blind"`                   // 盲注資訊
	ActionTimeSecs       int    `json:"action_time_secs"`        // 玩家動作思考時間 (秒數)
}

type Blind struct {
	ID               string       `json:"id"`                 // ID
	Name             string       `json:"name"`               // 名稱
	InitialLevel     int          `json:"initial_level"`      // 起始盲注級別
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
func (t *Table) RefreshUpdateAt() {
	t.UpdateAt = time.Now().Unix()
}

func (t *Table) Reset() {
	t.State.PlayingPlayerIndexes = []int{}
	for i := 0; i < len(t.State.PlayerStates); i++ {
		t.State.PlayerStates[i].Positions = make([]string, 0)
	}
}

func (t *Table) ConfigureWithSetting(setting TableSetting) {
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
				Level:        blindLevel.Level,
				SBChips:      blindLevel.SBChips,
				BBChips:      blindLevel.BBChips,
				AnteChips:    blindLevel.AnteChips,
				DurationMins: blindLevel.DurationMins,
				LevelEndAt:   UnsetValue,
			}
		}).([]*TableBlindLevelState),
	}
	state := TableState{
		GameCount:              0,
		StartGameAt:            UnsetValue,
		BlindState:             &blindState,
		CurrentDealerSeatIndex: UnsetValue,
		CurrentBBSeatIndex:     UnsetValue,
		PlayerSeatMap:          NewDefaultSeatMap(setting.CompetitionMeta.TableMaxSeatCount),
		PlayerStates:           make([]*TablePlayerState, 0),
		PlayingPlayerIndexes:   make([]int, 0),
		Status:                 TableStateStatus_TableGameCreated,
	}
	t.State = &state

	// handle auto join players
	if len(setting.JoinPlayers) > 0 {
		// auto join players
		t.State.PlayerStates = funk.Map(setting.JoinPlayers, func(p JoinPlayer) *TablePlayerState {
			return &TablePlayerState{
				PlayerID:          p.PlayerID,
				SeatIndex:         UnsetValue,
				Positions:         []string{Position_Unknown},
				IsParticipated:    true,
				IsBetweenDealerBB: false,
				Bankroll:          p.RedeemChips,
			}
		}).([]*TablePlayerState)

		// update seats
		for playerIdx := 0; playerIdx < len(t.State.PlayerStates); playerIdx++ {
			seatIdx := RandomSeatIndex(state.PlayerSeatMap)
			t.State.PlayerSeatMap[seatIdx] = playerIdx
			t.State.PlayerStates[playerIdx].SeatIndex = seatIdx
		}
	}
}

func (t *Table) ActivateBlindState() {
	for idx, levelState := range t.State.BlindState.LevelStates {
		if levelState.Level == t.State.BlindState.InitialLevel {
			t.State.BlindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := t.State.StartGameAt
	for i := (t.State.BlindState.InitialLevel - 1); i < len(t.State.BlindState.LevelStates); i++ {
		if i == t.State.BlindState.InitialLevel-1 {
			t.State.BlindState.LevelStates[i].LevelEndAt = blindStartAt
		} else {
			t.State.BlindState.LevelStates[i].LevelEndAt = t.State.BlindState.LevelStates[i-1].LevelEndAt
		}
		blindPassedSeconds := int64((time.Duration(t.State.BlindState.LevelStates[i].DurationMins) * time.Minute).Seconds())
		t.State.BlindState.LevelStates[i].LevelEndAt += blindPassedSeconds
	}
}

func (t *Table) GameOpen(gameEngine *GameEngine) error {
	// Step 1: 重設桌次狀態
	t.Reset()

	// Step 2: 檢查參賽資格
	for i := 0; i < len(t.State.PlayerStates); i++ {
		// 先讓沒有坐在 大盲、Dealer 之間的玩家參賽
		if t.State.PlayerStates[i].IsParticipated || t.State.PlayerStates[i].IsBetweenDealerBB {
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

	// Step 4: 計算新 Dealer SeatIndex & PlayerIndex
	newDealerPlayerIdx := FindDealerPlayerIndex(t.State.GameCount, t.State.CurrentDealerSeatIndex, t.Meta.CompetitionMeta.TableMinPlayingCount, t.Meta.CompetitionMeta.TableMaxSeatCount, t.State.PlayerStates, t.State.PlayerSeatMap)
	newDealerTableSeatIdx := t.State.PlayerStates[newDealerPlayerIdx].SeatIndex

	// Step 5: 處理玩家參賽狀態，確認玩家在 BB-Dealer 的參賽權
	for i := 0; i < len(t.State.PlayerStates); i++ {
		if !t.State.PlayerStates[i].IsBetweenDealerBB {
			continue
		}

		if newDealerTableSeatIdx-t.State.CurrentDealerSeatIndex < 0 {
			for j := t.State.CurrentDealerSeatIndex + 1; j < newDealerTableSeatIdx+t.Meta.CompetitionMeta.TableMaxSeatCount; j++ {
				if (j % t.Meta.CompetitionMeta.TableMaxSeatCount) != t.State.PlayerStates[i].SeatIndex {
					continue
				}

				t.State.PlayerStates[i].IsParticipated = true
				t.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		} else {
			for j := t.State.CurrentDealerSeatIndex + 1; j < newDealerTableSeatIdx; j++ {
				if j != t.State.PlayerStates[i].SeatIndex {
					continue
				}

				t.State.PlayerStates[i].IsParticipated = true
				t.State.PlayerStates[i].IsBetweenDealerBB = false
			}
		}
	}

	// Step 6: 計算 & 更新本手參與玩家的 PlayerIndex 陣列
	playingPlayerIndexes := FindPlayingPlayerIndexes(newDealerTableSeatIdx, t.State.PlayerSeatMap, t.State.PlayerStates)
	t.State.PlayingPlayerIndexes = playingPlayerIndexes

	// Step 7: 計算 & 更新本手參與玩家位置資訊
	positionMap := GetPlayerPositionMap(t.Meta.CompetitionMeta.Rule, t.State.PlayerStates, playingPlayerIndexes)
	for playerIdx := 0; playerIdx < len(t.State.PlayerStates); playerIdx++ {
		positions, exist := positionMap[playerIdx]
		if exist && t.State.PlayerStates[playerIdx].IsParticipated {
			t.State.PlayerStates[playerIdx].Positions = positions
		}
	}

	// Step 8: 更新桌次狀態 (GameCount, 當前 Dealer & BB 位置)
	t.State.GameCount = t.State.GameCount + 1
	t.State.CurrentDealerSeatIndex = newDealerTableSeatIdx
	if len(playingPlayerIndexes) == 2 {
		bbPlayerIdx := playingPlayerIndexes[1]
		t.State.CurrentBBSeatIndex = t.State.PlayerStates[bbPlayerIdx].SeatIndex
	} else if len(playingPlayerIndexes) > 2 {
		bbPlayerIdx := playingPlayerIndexes[2]
		t.State.CurrentBBSeatIndex = t.State.PlayerStates[bbPlayerIdx].SeatIndex
	} else {
		t.State.CurrentBBSeatIndex = UnsetValue
	}

	// Step 9: 更新當前桌次事件
	t.State.Status = TableStateStatus_TableGameMatchOpen

	// Step 10: 啟動本手遊戲引擎 & 更新遊戲狀態
	blind := *t.State.BlindState.LevelStates[t.State.BlindState.CurrentLevelIndex]
	dealerBlindTimes := t.Meta.CompetitionMeta.Blind.DealerBlindTimes
	gameEngineSetting := NewGameEngineSetting(t.Meta.CompetitionMeta.Rule, blind, dealerBlindTimes, t.State.PlayerStates, t.State.PlayingPlayerIndexes)
	if err := gameEngine.Start(gameEngineSetting); err != nil {
		return err
	}

	t.debugPrintTable(fmt.Sprintf("第 (%d) 手開局資訊", t.State.GameCount)) // TODO: test only, remove it later on
	return nil
}

func (t *Table) Settlement() {
	// Step 1: 把玩家輸贏籌碼更新到 Bankroll
	for _, player := range t.State.GameState.Result.Players {
		playerIdx := t.State.PlayingPlayerIndexes[player.Idx]
		t.State.PlayerStates[playerIdx].Bankroll = player.Final
	}

	// Step 2: 更新盲注 Level
	t.State.BlindState.Update()

	// Step 3: 依照桌次目前狀況更新事件
	if !t.State.BlindState.IsFinalBuyInLevel() && len(t.AlivePlayers()) < 2 {
		t.State.Status = TableStateStatus_TableGamePaused
	} else if t.State.BlindState.IsBreaking() {
		t.State.Status = TableStateStatus_TableGamePaused
	} else if t.IsClose(t.EndGameAt(), t.AlivePlayers(), t.State.BlindState.IsFinalBuyInLevel()) {
		t.State.Status = TableStateStatus_TableGameClosed
	}

	t.debugPrintGameStateResult() // TODO: test only, remove it later on
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
			SeatIndex:         UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          redeemChips,
		}
		t.State.PlayerStates = append(t.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(t.State.PlayerStates) - 1
		seatIdx := RandomSeatIndex(t.State.PlayerSeatMap)
		t.State.PlayerSeatMap[seatIdx] = newPlayerIdx
		t.State.PlayerStates[newPlayerIdx].SeatIndex = seatIdx
		t.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, t.State.CurrentDealerSeatIndex, t.State.CurrentBBSeatIndex, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		t.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(t.State.PlayerStates[targetPlayerIdx].SeatIndex, t.State.CurrentDealerSeatIndex, t.State.CurrentBBSeatIndex, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
		t.State.PlayerStates[targetPlayerIdx].Bankroll += redeemChips
		t.State.PlayerStates[targetPlayerIdx].IsParticipated = true
	}

	return nil
}

func (t *Table) PlayerRedeemChips(playerIdx int, redeemChips int64) {
	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	if t.State.PlayerStates[playerIdx].Bankroll == 0 {
		t.State.PlayerStates[playerIdx].IsBetweenDealerBB = IsBetweenDealerBB(t.State.PlayerStates[playerIdx].SeatIndex, t.State.CurrentDealerSeatIndex, t.State.CurrentBBSeatIndex, t.Meta.CompetitionMeta.TableMaxSeatCount, t.Meta.CompetitionMeta.Rule)
	}
	t.State.PlayerStates[playerIdx].Bankroll += redeemChips
}

func (t *Table) PlayersLeave(leavePlayerIndexes []int) {
	// set leave PlayerIdx int seatMap to UnsetValue
	leavePlayerIDMap := make(map[string]interface{})
	for _, leavePlayerIdx := range leavePlayerIndexes {
		leavePlayer := t.State.PlayerStates[leavePlayerIdx]
		leavePlayerIDMap[leavePlayer.PlayerID] = struct{}{}
		t.State.PlayerSeatMap[leavePlayer.SeatIndex] = UnsetValue
	}

	// delete target players in PlayerStates
	t.State.PlayerStates = funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*TablePlayerState)

	// update current PlayerSeatMap player indexes in PlayerSeatMap
	for newPlayerIdx, player := range t.State.PlayerStates {
		t.State.PlayerSeatMap[player.SeatIndex] = newPlayerIdx
	}
}

// Table Getters
func (t Table) ModeRule() string {
	return fmt.Sprintf("%s_%s_holdem", t.Meta.CompetitionMeta.Mode, t.Meta.CompetitionMeta.Rule)
}

func (t Table) GetJSON() (*string, error) {
	encoded, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	json := string(encoded)
	return &json, nil
}

func (t Table) ParticipatedPlayers() []*TablePlayerState {
	return funk.Filter(t.State.PlayerStates, func(player *TablePlayerState) bool {
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

func (t Table) PlayingPlayerIndex(playerID string) int {
	playerIdx := UnsetValue
	for idx, player := range t.State.PlayerStates {
		if player.PlayerID == playerID {
			playerIdx = idx
			break
		}
	}

	if playerIdx == UnsetValue || !funk.Contains(t.State.PlayingPlayerIndexes, playerIdx) {
		return UnsetValue
	}
	return playerIdx
}

/*
	isTableClose 計算本桌是否已結束
	  - 結束條件 1: 達到賽局結束時間
	  - 結束條件 2: 停止買入後且存活玩家剩餘 1 人
*/
func (t Table) IsClose(endGameAt int64, alivePlayers []*TablePlayerState, isFinalBuyInLevel bool) bool {
	return time.Now().Unix() > endGameAt || (isFinalBuyInLevel && len(alivePlayers) == 1)
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
	if bs.FinalBuyInLevelIndex == UnsetValue {
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

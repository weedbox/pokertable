package pokertable

import (
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

type TablePlayerGameStatistics struct {
	ActionTimes int    `json:"action_times"` // 每手下注動作總次數
	RaiseTimes  int    `json:"raise_times"`  // 每手加注總次數
	CallTimes   int    `json:"call_times"`   // 每手跟注總次數
	CheckTimes  int    `json:"check_times"`  // 每手過牌總次數
	IsFold      bool   `json:"is_fold"`      // 每手是否蓋牌
	FoldRound   string `json:"fold_round"`   // 每手蓋牌回合

	// VPIP
	IsVPIP bool `json:"is_vpip"` // 每手是否 VPIP

	// PFR
	PFRTimes int `json:"pfr_times"` // 每手翻前加注總次數

	// ATS
	// TODO: not yet implemented
	IsATS      bool `json:"is_ats"`      // 每手輪到動作時是否可以 ATS
	ATSChances int  `json:"ats_chances"` // 每手偷盲機會
	ATSTimes   int  `json:"ats_times"`   // 每手偷盲次數

	// 3-Bet
	// TODO: not yet implemented
	IsThreeB      bool `json:"is_three_b"`      // 每手輪到動作時是否可以 3-Bet
	ThreeBChances int  `json:"three_b_chances"` // 每手 3-Bet 機會
	ThreeBTimes   int  `json:"three_b_times"`   // 每手 3-Bet 次數
}

// func (te *tableEngine) initPlayerGameStatistics(playerID string) {
// 	te.table.State.GameStatistics.Players[playerID] = &PlayerGameStatistics{
// 		// TODO: implement this (default values)
// 	}
// }

// func (te *tableEngine) removePlayerGameStatistics(playerID string) {
// 	delete(te.table.State.GameStatistics.Players, playerID)
// }

// func (te *tableEngine) resetPlayerGameStatistics(playerIDs []string) {
// 	for _, playerID := range playerIDs {
// 		pgs, exist := te.table.State.GameStatistics.Players[playerID]
// 		if !exist {
// 			fmt.Printf("[DEBUG#resetPlayerGameStatistics] player (%s) is not in the GameStatistics.Players", playerID)
// 			return
// 		}

// 		pgs.VPIPData.IsVPIP = false
// 	}
// }

/*
VPIP: 显示翻牌前主动住Pot里面投注的比例。(https://steemit.com/cn/@davidfnck/poker-tracker-vpip-pfr-af)
- 計算方式: preflop 時還不是 VPIP 玩家且有做任何 Raise、Call、Bet、Allin 動作
*/
func (te *tableEngine) updatePlayerVPIP(playerIdx int, action string, gs *pokerface.GameState) {
	if gs.Status.Round != GameRound_Preflop {
		return
	}

	playerState := te.table.State.PlayerStates[playerIdx]
	if playerState.GameStatistics.IsVPIP {
		return
	}

	vpipActions := []string{
		WagerAction_Bet,
		WagerAction_Call,
		WagerAction_Raise,
		WagerAction_AllIn,
	}
	isVPIP := funk.Contains(vpipActions, action)
	if isVPIP {
		playerState.GameStatistics.IsVPIP = true
	} else {
		playerState.GameStatistics.IsVPIP = false
	}
}

/*
PFR: 是指玩家翻牌前加注的频率。该数据仅计算你在翻牌前加注入池的次数，并不包括翻牌前的跛入(limp)和跟注。(https://www.legendpoker.cn/xw/zx/3244.html)
- 計算方式: preflop 時玩家動作為 Raise 或 Allin (且是 raiser)
*/
func (te *tableEngine) updatePlayerPFRTimes(playerIdx int, gs *pokerface.GameState) {
	if gs.Status.Round != GameRound_Preflop {
		return
	}

	playerState := te.table.State.PlayerStates[playerIdx]
	playerState.GameStatistics.PFRTimes++
}

func (te *tableEngine) batchUpdatePlayerGameStatistics(gs *pokerface.GameState) {
	te.lock.Lock()
	defer te.lock.Unlock()

	// for _, p := range gs.Players {
	// 	playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(p.Idx)
	// 	if playerIdx == UnsetValue {
	// 		fmt.Printf("[DEBUG#batchUpdatePlayerGameStatistics] can't find player index from game player index (%d)", p.Idx)
	// 		continue
	// 	}

	// 	player := te.table.State.PlayerStates[playerIdx]
	// 	pgs, exist := te.table.State.GameStatistics.Players[player.PlayerID]
	// 	if !exist {
	// 		fmt.Printf("[DEBUG#batchUpdatePlayerGameStatistics] player (%s) is not in the GameStatistics.Players", player.PlayerID)
	// 		continue
	// 	}

	// }
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

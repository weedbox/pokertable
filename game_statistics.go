package pokertable

import (
	"fmt"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
)

const (
	// Preflop GameStatistics
	GameStatisticRound_VPIP     = "vpip"
	GameStatisticRound_PFR      = "pfr"
	GameStatisticRound_ATS      = "ats"
	GameStatisticRound_ThreeBet = "three-bet"
	GameStatisticRound_Ft3B     = "ft3b"

	// Postlop GameStatistics
	GameStatisticRound_CheckRaise = "check-raise"
	GameStatisticRound_CBet       = "c-bet"
	GameStatisticRound_FtCB       = "ftcb"
)

type TablePlayerGameStatistics struct {
	ActionTimes int    `json:"action_times"` // 每手下注動作總次數
	RaiseTimes  int    `json:"raise_times"`  // 每手加注總次數
	CallTimes   int    `json:"call_times"`   // 每手跟注總次數
	CheckTimes  int    `json:"check_times"`  // 每手過牌總次數
	IsFold      bool   `json:"is_fold"`      // 每手是否蓋牌
	FoldRound   string `json:"fold_round"`   // 每手蓋牌回合

	// preflop: VPIP
	IsVPIPChance bool `json:"is_vpip_chance"` // 每手翻前 VPIP 機會
	IsVPIP       bool `json:"is_vpip"`        // 每手翻前是否有過 VPIP

	// preflop: PFR
	IsPFRChance bool `json:"is_pfr_chance"` // 每手翻前 PFR 機會
	IsPFR       bool `json:"is_pfr"`        // 每手翻前是否有過 PFR

	// preflop: ATS
	IsATSChance bool `json:"is_ats_chance"` // 每手翻前 ATS 機會
	IsATS       bool `json:"is_ats"`        // 每手翻前否有過 ATS

	// preflop: 3-Bet
	Is3BChance bool `json:"is_3b_chance"` // 每手翻前 3-Bet 機會
	Is3B       bool `json:"is_3b"`        // 每手翻前否有過 3-Bet

	// preflop: Ft3B
	IsFt3BChance bool `json:"is_ft3b_chance"` // 每手翻前 Ft3B 機會
	IsFt3B       bool `json:"is_ft3b"`        // 每手翻前否有過 Ft3B

	// flop: C/R TODO: flop/turn/river 都要
	IsCheckRaiseChance bool `json:"is_check_raise_chance"` // 每手翻前 C/R 機會
	IsCheckRaise       bool `json:"is_check_raise"`        // 每手翻前否有過 C/R

	// flop: C-Bet
	IsCBetChance bool `json:"is_c_bet_chance"` // 每手翻前 C-Bet 機會
	IsCBet       bool `json:"is_c_bet"`        // 每手翻前否有過 C-Bet

	// flop: FtCB
	IsFtCBChance bool `json:"is_ftcb_chance"` // 每手翻前 FtCB 機會
	IsFtCB       bool `json:"is_ftcb"`        // 每手翻前否有過 FtCB

	// settle
	ShowdownWinningChance bool `json:"showdown_winning_chance"` // 每手結算時 Showdown Winning 機會
	IsShowdownWinning     bool `json:"is_showdown_winning"`     // 每手結算時是否 Showdown Winning
}

func NewPlayerGameStatistics() TablePlayerGameStatistics {
	return TablePlayerGameStatistics{
		ActionTimes: 0,
		RaiseTimes:  0,
		CallTimes:   0,
		CheckTimes:  0,
		IsFold:      false,
		FoldRound:   "",

		// preflop: VPIP
		IsVPIPChance: false,
		IsVPIP:       false,

		// preflop: PFR
		IsPFRChance: false,
		IsPFR:       false,

		// preflop: ATS
		IsATSChance: false,
		IsATS:       false,

		// preflop: 3-Bet
		Is3BChance: false,
		Is3B:       false,

		// preflop: Fold to 3-Bet
		IsFt3BChance: false,
		IsFt3B:       false,

		// postflop: C/R
		IsCheckRaiseChance: false,
		IsCheckRaise:       false,

		// C-Bet
		IsCBetChance: false,
		IsCBet:       false,

		// Fold to C-Bet
		IsFtCBChance: false,
		IsFtCB:       false,

		// settle
		ShowdownWinningChance: false,
		IsShowdownWinning:     false,
	}
}

func (te *tableEngine) refreshThreeBet(playerState *TablePlayerState, playerIdx int) {
	// 在有玩家 3-Bet 的情況下，其他玩家 Raise 會重設該玩家 3-Bet 標籤
	hasThreeBet := false
	for _, p := range te.table.State.PlayerStates {
		if p.GameStatistics.Is3B {
			hasThreeBet = true
			break
		}
	}
	if hasThreeBet {
		for i := 0; i < len(te.table.State.PlayerStates); i++ {
			te.table.State.PlayerStates[i].GameStatistics.Is3B = false
		}
	}

	if playerState.GameStatistics.Is3BChance {
		// 整桌只會有一個玩家有 3-Bet 標籤
		for i := 0; i < len(te.table.State.PlayerStates); i++ {
			if i == playerIdx {
				te.table.State.PlayerStates[i].GameStatistics.Is3B = true
			} else {
				te.table.State.PlayerStates[i].GameStatistics.Is3B = false
			}
		}
	}
}

func (te *tableEngine) updateCurrentPlayerGameStatistics(gs *pokerface.GameState) {
	te.lock.Lock()
	defer te.lock.Unlock()

	// check current player
	currentGamePlayerIdx := gs.Status.CurrentPlayer
	currentPlayerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(currentGamePlayerIdx)
	if currentPlayerIdx == UnsetValue {
		fmt.Printf("[DEBUG#updateCurrentPlayerGameStatistics] can't find current player index from game player index (%d)", currentGamePlayerIdx)
	} else {
		currentPlayer := te.table.State.PlayerStates[currentPlayerIdx]

		// 計算 VPIP
		if te.isVPIPChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.IsVPIPChance = true
		}

		// 計算 PFR
		if te.isPFRChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.IsPFRChance = true
		}

		// 計算 ATS
		if te.isATSChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.IsATSChance = true
		}

		// 計算 3-Bet
		if te.is3BChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.Is3BChance = true
		}

		// 計算 Ft3B
		if te.IsFt3BChance(currentGamePlayerIdx, te.table.State.PlayerStates, gs) {
			currentPlayer.GameStatistics.IsFt3BChance = true
		}

		// 計算 C/R
		if te.isCheckRaiseChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.IsCheckRaiseChance = true
		}

		// 計算 C-Bet
		if te.isCBetChance(currentGamePlayerIdx, gs) {
			currentPlayer.GameStatistics.IsCBetChance = true
		}

		// 計算 FtCB
		if te.isFtCBChance(currentGamePlayerIdx, te.table.State.PlayerStates, gs) {
			currentPlayer.GameStatistics.IsFtCBChance = true
		}
	}
}

// isVPIPChance: preflop 時還沒入池 (not VPIP)
func (te *tableEngine) isVPIPChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_VPIP) {
		return false
	}

	playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(gamePlayerIdx)
	if playerIdx == UnsetValue {
		fmt.Printf("[DEBUG#isVPIPChance] can't find player index from game player index (%d)", gamePlayerIdx)
		return false
	}

	if !te.table.State.PlayerStates[playerIdx].GameStatistics.IsVPIP {
		return true
	}

	return false
}

// isPFRChance: preflop 時，並且前位玩家皆跟注或棄牌
func (te *tableEngine) isPFRChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_PFR) {
		return false
	}

	allinCall := 0
	call := 0
	fold := 0
	for _, p := range gs.Players {
		if gamePlayerIdx == p.Idx {
			continue
		}

		if p.DidAction == WagerAction_AllIn {
			if gs.Status.CurrentRaiser != p.Idx {
				allinCall++
			}
		}

		if p.DidAction == WagerAction_Call {
			call++
		}

		if p.DidAction == WagerAction_Fold {
			fold++
		}
	}

	return (allinCall + call + fold) == len(gs.Players)-1
}

/*
isATSChance preflop 時，SB/CO/Dealer 玩家在前位已行動玩家皆棄牌
*/
func (te *tableEngine) isATSChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_ATS) {
		return false
	}

	// 計算除自己位外的已行動玩家是否都 Fold
	fold := 0
	acted := 0
	for _, p := range gs.Players {
		if p.Acted {
			acted++

			if gamePlayerIdx != p.Idx && p.Fold {
				fold++
			}
		}
	}

	validPositions := gs.HasPosition(gamePlayerIdx, Position_SB) || gs.HasPosition(gamePlayerIdx, Position_CO) || gs.HasPosition(gamePlayerIdx, Position_Dealer)
	if (fold == acted-1) && validPositions {
		return true
	}

	return false
}

// is3BChance: preflop 時前位只有一位玩家進行加注，且其餘玩家皆跟注或棄牌
func (te *tableEngine) is3BChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_ThreeBet) {
		return false
	}

	allinRaiser := 0
	raiser := 0
	for _, p := range gs.Players {
		if gamePlayerIdx == p.Idx {
			continue
		}

		if p.DidAction == WagerAction_AllIn && gs.Status.CurrentRaiser == p.Idx {
			allinRaiser++
		}

		if p.DidAction == WagerAction_Raise {
			raiser++
		}
	}

	// 只有一位玩家 Allin (Raiser) or 只有一位玩家 Raise 才符合條件
	if (allinRaiser == 1 && raiser == 0) || (allinRaiser == 0 && raiser == 1) {
		return true
	}

	return false
}

// IsFt3BChance: preflop 時當玩家在加注或跟注後遇到其他玩家的3-Bet（Re-raise）
func (te *tableEngine) IsFt3BChance(gamePlayerIdx int, players []*TablePlayerState, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_Ft3B) {
		return false
	}

	playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(gamePlayerIdx)
	if playerIdx == UnsetValue {
		fmt.Printf("[DEBUG#IsFt3BChance] can't find player index from game player index (%d)", gamePlayerIdx)
		return false
	}

	for idx, p := range players {
		if playerIdx == idx {
			continue
		}

		if p.GameStatistics.Is3B {
			return true
		}
	}

	return false
}

func (te *tableEngine) isCheckRaiseChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_CheckRaise) {
		return false
	}

	player := gs.GetPlayer(gamePlayerIdx)

	// 自己要先 check 過
	if player.DidAction != WagerAction_Check {
		return false
	}

	// 自己要可以 Raise or Allin (raiser): 後手/剩餘籌碼 > MiniBet
	canRaise := funk.Contains(player.AllowedActions, WagerAction_Raise)
	canAllinRaiser := funk.Contains(player.AllowedActions, WagerAction_AllIn) && player.StackSize > gs.Status.MiniBet
	if canRaise || canAllinRaiser {
		return true
	}

	return false
}

func (te *tableEngine) isCBetChance(gamePlayerIdx int, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_CBet) {
		return false
	}

	// 自己在 preflop 時要是 raiser 且有下列任一動作: Bet or Raise or Allin (raiser): 後手/剩餘籌碼 > MiniBet
	player := gs.GetPlayer(gamePlayerIdx)
	isPreflopRaiser := gs.Status.CurrentRaiser == gamePlayerIdx
	canBet := funk.Contains(player.AllowedActions, WagerAction_Bet)
	canRaise := funk.Contains(player.AllowedActions, WagerAction_Raise)
	canAllinRaiser := funk.Contains(player.AllowedActions, WagerAction_AllIn) && player.StackSize > gs.Status.MiniBet
	validAction := canBet || canRaise || canAllinRaiser

	if isPreflopRaiser && validAction {
		return true
	}

	return false
}

func (te *tableEngine) isFtCBChance(gamePlayerIdx int, players []*TablePlayerState, gs *pokerface.GameState) bool {
	if !te.validateGameStatisticGameState(gamePlayerIdx, gs) {
		return false
	}

	if !te.validateGameRoundChance(gs.Status.Round, GameStatisticRound_FtCB) {
		return false
	}

	playerIdx := te.table.FindPlayerIndexFromGamePlayerIndex(gamePlayerIdx)
	if playerIdx == UnsetValue {
		fmt.Printf("[DEBUG#isFtCBChance] can't find player index from game player index (%d)", gamePlayerIdx)
		return false
	}

	for idx, p := range players {
		if playerIdx == idx {
			continue
		}

		if p.GameStatistics.IsCBet {
			return true
		}
	}

	return false
}

func (te *tableEngine) validateGameStatisticGameState(gamePlayerIdx int, gs *pokerface.GameState) bool {
	validEvent := pokerface.GameEventSymbols[pokerface.GameEvent_Started]
	validRounds := []string{
		GameRound_Preflop,
		GameRound_Flop,
		GameRound_Turn,
		GameRound_River,
	}
	validActions := []string{
		WagerAction_Fold,
		WagerAction_Check,
		WagerAction_Call,
		WagerAction_AllIn,
		WagerAction_Bet,
		WagerAction_Raise,
	}

	if !(gs.Status.CurrentEvent == validEvent && funk.Contains(validRounds, gs.Status.Round)) {
		return false
	}

	player := gs.Players[gamePlayerIdx]
	if !player.Acted {
		return false
	}

	if len(player.AllowedActions) <= 0 {
		return false
	}

	for _, action := range player.AllowedActions {
		if !funk.Contains(validActions, action) {
			return false
		}
	}

	return true
}

func (te *tableEngine) validateGameRoundChance(round, statisticRound string) bool {
	preflopChances := []string{
		GameStatisticRound_VPIP,
		GameStatisticRound_PFR,
		GameStatisticRound_ATS,
		GameStatisticRound_ThreeBet,
		GameStatisticRound_Ft3B,
	}
	flopChances := []string{
		GameStatisticRound_CheckRaise,
		GameStatisticRound_CBet,
		GameStatisticRound_FtCB,
	}

	if round == GameRound_Preflop {
		return funk.Contains(preflopChances, statisticRound)
	} else if round == GameRound_Flop {
		return funk.Contains(flopChances, statisticRound)
	} else {
		return false
	}
}

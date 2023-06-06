package util

const (
	// General
	UnsetValue = -1

	// Position
	Position_Unknown = "unknown"
	Position_Dealer  = "dealer"
	Position_SB      = "sb"
	Position_BB      = "bb"
	Position_UG      = "ug"
	Position_UG1     = "ug1"
	Position_UG2     = "ug2"
	Position_UG3     = "ug3"
	Position_HJ      = "hj"
	Position_CO      = "co"

	// Wager Action
	WagerAction_Fold  = "fold"
	WagerAction_Check = "check"
	WagerAction_Call  = "call"
	WagerAction_AllIn = "allin"
	WagerAction_Bet   = "Bet"
	WagerAction_Raise = "raise"

	// Round
	GameRound_Preflod = "preflop"
	GameRound_Flod    = "flop"
	GameRound_Turn    = "turn"
	GameRound_River   = "river"
)

package pokertable

const (
	// General
	UnsetValue = -1

	// CompetitionMode
	CompetitionMode_CT   = "ct"   // 倒數錦標賽
	CompetitionMode_MTT  = "mtt"  // 大型錦標賽
	CompetitionMode_Cash = "cash" // 現金桌

	// CompetitionRule
	CompetitionRule_Default   = "default"    // 常牌
	CompetitionRule_ShortDeck = "short_deck" // 短牌
	CompetitionRule_Omaha     = "omaha"      // 奧瑪哈

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

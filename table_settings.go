package pokertable

type TableSetting struct {
	ShortID         string          `json:"short_id"`
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	InvitationCode  string          `json:"invitation_code"`
	CompetitionMeta CompetitionMeta `json:"competition_meta"`
	JoinPlayers     []JoinPlayer    `json:"join_players"`
}

type JoinPlayer struct {
	PlayerID    string `json:"player_id"`
	RedeemChips int64  `json:"redeem_chips"`
	Seat        int    `json:"seat"`
}

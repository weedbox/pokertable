package pokertable

type TableSetting struct {
	TableID     string       `json:"table_id"`
	Meta        TableMeta    `json:"table_meta"`
	JoinPlayers []JoinPlayer `json:"join_players"`
}

type JoinPlayer struct {
	PlayerID    string `json:"player_id"`
	RedeemChips int64  `json:"redeem_chips"`
	Seat        int    `json:"seat"`
}

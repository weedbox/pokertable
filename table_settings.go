package pokertable

import (
	"github.com/weedbox/pokerface"
)

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
}

type GameEngineSetting struct {
	Rule    string
	Ante    int64
	Blind   pokerface.BlindSetting
	Players []*pokerface.PlayerSetting
}

package pokertable

import "github.com/weedbox/pokerface"

type GameEngineSetting struct {
	Rule    string
	Ante    int64
	Blind   pokerface.BlindSetting
	Players []*pokerface.PlayerSetting
}

func NewGameEngineSetting(rule string, blind TableBlindLevelState, dealerBlindTimes int, players []*TablePlayerState, playingPlayerIndexes []int) GameEngineSetting {
	setting := GameEngineSetting{
		Rule: rule,
		Ante: blind.AnteChips,
		Blind: pokerface.BlindSetting{
			Dealer: blind.AnteChips * (int64(dealerBlindTimes) - 1),
			SB:     blind.SBChips,
			BB:     blind.BBChips,
		},
	}
	playerSettings := make([]*pokerface.PlayerSetting, 0)
	for _, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		playerSettings = append(playerSettings, &pokerface.PlayerSetting{
			Bankroll:  player.Bankroll,
			Positions: player.Positions,
		})
	}
	setting.Players = playerSettings
	return setting
}

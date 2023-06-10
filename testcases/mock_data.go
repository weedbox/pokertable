package testcases

import (
	"github.com/google/uuid"
	"github.com/weedbox/pokertable"
)

func NewDefaultTableSetting(joinPlayers ...pokertable.JoinPlayer) pokertable.TableSetting {
	return pokertable.TableSetting{
		ShortID:        "ABC123",
		Code:           "01",
		Name:           "table name",
		InvitationCode: "come_to_play",
		CompetitionMeta: pokertable.CompetitionMeta{
			ID: "competition id",
			Blind: pokertable.Blind{
				ID:               uuid.New().String(),
				Name:             "blind name",
				InitialLevel:     1,
				FinalBuyInLevel:  2,
				DealerBlindTimes: 1,
				Levels: []pokertable.BlindLevel{
					{
						Level:        1,
						SBChips:      10,
						BBChips:      20,
						AnteChips:    0,
						DurationMins: 10,
					},
					{
						Level:        2,
						SBChips:      20,
						BBChips:      30,
						AnteChips:    0,
						DurationMins: 10,
					},
					{
						Level:        3,
						SBChips:      30,
						BBChips:      40,
						AnteChips:    0,
						DurationMins: 10,
					},
				},
			},
			MaxDurationMins:      60,
			Rule:                 pokertable.CompetitionRule_Default,
			Mode:                 pokertable.CompetitionMode_CT,
			TableMaxSeatCount:    9,
			TableMinPlayingCount: 2,
			MinChipsUnit:         10,
			ActionTimeSecs:       10,
		},
		JoinPlayers: joinPlayers,
	}
}

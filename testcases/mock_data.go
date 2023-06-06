package testcases

import (
	"github.com/google/uuid"
	"github.com/weedbox/pokertable/model"
	"github.com/weedbox/pokertable/util"
)

func NewDefaultTableSetting(joinPlayers ...model.JoinPlayer) model.TableSetting {
	return model.TableSetting{
		ShortID:           "ABC123",
		Code:              "01",
		Name:              "table name",
		InvitationCode:    "come_to_play",
		BlindInitialLevel: 1,
		CompetitionMeta: model.CompetitionMeta{
			Blind: model.Blind{
				ID:              uuid.New().String(),
				Name:            "blind name",
				FinalBuyInLevel: 2,
				Levels: []model.BlindLevel{
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
			Rule:                 util.CompetitionRule_Default,
			Mode:                 util.CompetitionMode_MTT,
			TableMaxSeatCount:    9,
			TableMinPlayingCount: 2,
			MinChipsUnit:         10,
		},
		JoinPlayers: joinPlayers,
	}
}

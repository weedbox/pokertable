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
				ID:              uuid.New().String(),
				Name:            "blind name",
				InitialLevel:    1,
				FinalBuyInLevel: 2,
				DealerBlindTime: 1,
				Levels: []pokertable.BlindLevel{
					{
						Level:    1,
						SB:       10,
						BB:       20,
						Ante:     0,
						Duration: 1,
					},
					{
						Level:    2,
						SB:       20,
						BB:       30,
						Ante:     0,
						Duration: 1,
					},
					{
						Level:    3,
						SB:       30,
						BB:       40,
						Ante:     0,
						Duration: 1,
					},
				},
			},
			MaxDuration:         3,
			Rule:                pokertable.CompetitionRule_Default,
			Mode:                pokertable.CompetitionMode_CT,
			TableMaxSeatCount:   9,
			TableMinPlayerCount: 2,
			MinChipUnit:         10,
			ActionTime:          10,
		},
		JoinPlayers: joinPlayers,
	}
}

package testcases

import (
	"github.com/google/uuid"
	"github.com/weedbox/pokermodel"
	"github.com/weedbox/pokertable/model"
)

func NewDefaultTableSetting(joinPlayers ...model.JoinPlayer) model.TableSetting {
	return model.TableSetting{
		ShortID:           "ABC123",
		Code:              "01",
		Name:              "table name",
		InvitationCode:    "come_to_play",
		BlindInitialLevel: 1,
		CompetitionMeta: pokermodel.CompetitionMeta{
			Blind: pokermodel.Blind{
				ID:              uuid.New().String(),
				Name:            "blind name",
				FinalBuyInLevel: 2,
				Levels: []pokermodel.BlindLevel{
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
			Ticket: pokermodel.Ticket{
				ID:   uuid.New().String(),
				Name: "ticket name",
			},
			Scene:           "scene 1",
			MaxDurationMins: 60,
			MinPlayerCount:  10,
			MaxPlayerCount:  100,
			Rule:            pokermodel.CompetitionRule_Default,
			Mode:            pokermodel.CompetitionMode_MTT,
			BuyInSetting: pokermodel.BuyInSetting{
				IsFree:     false,
				MinTickets: 1,
				MaxTickets: 1,
			},
			ReBuySetting: pokermodel.ReBuySetting{
				MinTicket: 1,
				MaxTicket: 1,
				MaxTimes:  5,
			},
			AddonSetting: pokermodel.AddonSetting{
				IsBreakOnly: true,
				RedeemChips: []int64{1000, 1100, 1200},
				MaxTimes:    3,
			},
			ActionTimeSecs:       10,
			TableMaxSeatCount:    9,
			TableMinPlayingCount: 2,
			MinChipsUnit:         10,
		},
		JoinPlayers: joinPlayers,
	}
}

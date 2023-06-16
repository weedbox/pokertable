package pokertable

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateTable(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	tableSettings := []TableSetting{
		NewDefaultTableSetting(
			JoinPlayer{PlayerID: "player 1", RedeemChips: 1000},
			JoinPlayer{PlayerID: "player 2", RedeemChips: 1000},
		),
		NewDefaultTableSetting(),
	}

	for _, tableSetting := range tableSettings {
		table, err := tableEngine.CreateTable(tableSetting)

		assert.Nil(t, err)
		assert.NotZero(t, table.ID)
		assert.NotZero(t, table.Meta)
		assert.NotZero(t, table.State)
		assert.Equal(t, TableStateStatus_TableCreated, table.State.Status)
		assert.Equal(t, len(tableSetting.JoinPlayers), len(table.State.PlayerStates))
		seatTakenCount := 0
		for _, playerIdx := range table.State.SeatMap {
			if playerIdx != -1 {
				seatTakenCount++
			}
		}
		assert.Equal(t, len(tableSetting.JoinPlayers), seatTakenCount)
		assert.NotZero(t, table.UpdateAt)
	}
}

func TestStartTableGame(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	tableSetting := NewDefaultTableSetting(
		JoinPlayer{PlayerID: "Jeffrey", RedeemChips: 1000},
		JoinPlayer{PlayerID: "Chuck", RedeemChips: 1000},
		JoinPlayer{PlayerID: "Fred", RedeemChips: 1000},
	)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	err = tableEngine.StartTableGame(table.ID)

	assert.Nil(t, err)
	assert.NotEqual(t, -1, table.State.StartAt)
	assert.NotEqual(t, -1, table.State.BlindState.CurrentLevelIndex)
	for _, blindLevel := range table.State.BlindState.LevelStates {
		assert.NotEqual(t, -1, blindLevel.EndAt)
	}
	assert.Equal(t, TableStateStatus_TableGamePlaying, table.State.Status)
	assert.Greater(t, table.State.GameCount, 0)
	assert.NotZero(t, table.State.GameState)
}

func TestPlayerJoin_BuyIn(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	joinPlayers := []JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 1000},
		{PlayerID: "Chuck", RedeemChips: 1000},
		{PlayerID: "Fred", RedeemChips: 1000},
	}

	for _, joinPlayer := range joinPlayers {
		err = tableEngine.PlayerJoin(table.ID, joinPlayer)
		assert.Nil(t, err)
	}

	assert.Equal(t, len(joinPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.SeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, len(joinPlayers), seatTakenCount)
}

func TestPlayerJoin_ReBuy(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	initialPlayers := []JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 0},
		{PlayerID: "Chuck", RedeemChips: 1000},
		{PlayerID: "Fred", RedeemChips: 1000},
	}
	tableSetting := NewDefaultTableSetting(initialPlayers...)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	reBuyPlayer := initialPlayers[0]
	reBuyPlayer.RedeemChips = 2000
	err = tableEngine.PlayerJoin(table.ID, reBuyPlayer)
	assert.Nil(t, err)

	assert.Equal(t, len(initialPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.SeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, len(initialPlayers), seatTakenCount)

	for _, player := range table.State.PlayerStates {
		if player.PlayerID == reBuyPlayer.PlayerID {
			assert.Equal(t, reBuyPlayer.RedeemChips, player.Bankroll)
		}
	}
}

func TestPlayerRedeemChips(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	initialPlayers := []JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 1000},
		{PlayerID: "Chuck", RedeemChips: 1000},
		{PlayerID: "Fred", RedeemChips: 1000},
	}
	tableSetting := NewDefaultTableSetting(initialPlayers...)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	addonPlayer := initialPlayers[0]
	addonPlayer.RedeemChips = 2000
	expectedAddonPlayerBankroll := initialPlayers[0].RedeemChips + addonPlayer.RedeemChips
	err = tableEngine.PlayerJoin(table.ID, addonPlayer)
	assert.Nil(t, err)

	assert.Equal(t, len(initialPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.SeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, len(initialPlayers), seatTakenCount)

	for _, player := range table.State.PlayerStates {
		if player.PlayerID == addonPlayer.PlayerID {
			assert.Equal(t, expectedAddonPlayerBankroll, player.Bankroll)
		}
	}
}

func TestPlayerLeave(t *testing.T) {
	tableEngine := NewTableEngine()
	tableEngine.OnTableUpdated(func(table *Table) {})
	initialPlayers := []JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 1000},
		{PlayerID: "Chuck", RedeemChips: 0},
		{PlayerID: "Fred", RedeemChips: 1000},
	}
	tableSetting := NewDefaultTableSetting(initialPlayers...)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	leavePlayerIDs := []string{initialPlayers[1].PlayerID}
	expectedPlayerCount := len(initialPlayers) - len(leavePlayerIDs)

	err = tableEngine.PlayersLeave(table.ID, leavePlayerIDs)
	assert.Nil(t, err)

	assert.Equal(t, expectedPlayerCount, len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.SeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, expectedPlayerCount, seatTakenCount)
}

func NewDefaultTableSetting(joinPlayers ...JoinPlayer) TableSetting {
	return TableSetting{
		ShortID:        "ABC123",
		Code:           "01",
		Name:           "table name",
		InvitationCode: "come_to_play",
		CompetitionMeta: CompetitionMeta{
			ID: "competition id",
			Blind: Blind{
				ID:              uuid.New().String(),
				Name:            "blind name",
				InitialLevel:    1,
				FinalBuyInLevel: 2,
				Levels: []BlindLevel{
					{
						Level:    1,
						SB:       10,
						BB:       20,
						Ante:     0,
						Duration: 10 * 60,
					},
					{
						Level:    2,
						SB:       20,
						BB:       30,
						Ante:     0,
						Duration: 10 * 60,
					},
					{
						Level:    3,
						SB:       30,
						BB:       40,
						Ante:     0,
						Duration: 10 * 60,
					},
				},
			},
			MaxDuration:         60 * 60,
			Rule:                CompetitionRule_Default,
			Mode:                CompetitionMode_MTT,
			TableMaxSeatCount:   9,
			TableMinPlayerCount: 2,
			MinChipUnit:         10,
			ActionTime:          10,
		},
		JoinPlayers: joinPlayers,
	}
}

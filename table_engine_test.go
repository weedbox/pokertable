package pokertable

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable/model"
	"github.com/weedbox/pokertable/util"
)

func TestCreateTable(t *testing.T) {
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	tableSettings := []model.TableSetting{
		NewDefaultTableSetting(
			model.JoinPlayer{PlayerID: "player 1", RedeemChips: 1000},
			model.JoinPlayer{PlayerID: "player 2", RedeemChips: 1000},
		),
		NewDefaultTableSetting(),
	}

	for _, tableSetting := range tableSettings {
		table, err := tableEngine.CreateTable(tableSetting)

		assert.Nil(t, err)
		assert.NotZero(t, table.ID)
		assert.NotZero(t, table.Meta)
		assert.NotZero(t, table.State)
		assert.Equal(t, model.TableStateStatus_TableGameCreated, table.State.Status)
		assert.Equal(t, len(tableSetting.JoinPlayers), len(table.State.PlayerStates))
		seatTakenCount := 0
		for _, playerIdx := range table.State.PlayerSeatMap {
			if playerIdx != -1 {
				seatTakenCount++
			}
		}
		assert.Equal(t, len(tableSetting.JoinPlayers), seatTakenCount)
		assert.NotZero(t, table.UpdateAt)
	}
}

func TestCloseTable(t *testing.T) {
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	expectedStatus := []model.TableStateStatus{
		model.TableStateStatus_TableGameAutoEnded,
		model.TableStateStatus_TableGameKilled,
	}

	for _, expectedStatus := range expectedStatus {
		table, err := tableEngine.CreateTable(NewDefaultTableSetting())
		table = tableEngine.CloseTable(table, expectedStatus)

		assert.Nil(t, err)
		assert.Equal(t, expectedStatus, table.State.Status)
	}
}

func TestStartGame(t *testing.T) {
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	tableSetting := NewDefaultTableSetting(
		model.JoinPlayer{PlayerID: "Jeffrey", RedeemChips: 1000},
		model.JoinPlayer{PlayerID: "Chuck", RedeemChips: 1000},
		model.JoinPlayer{PlayerID: "Fred", RedeemChips: 1000},
	)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	table, err = tableEngine.StartGame(table)

	assert.Nil(t, err)
	assert.NotEqual(t, -1, table.State.StartGameAt)
	assert.NotEqual(t, -1, table.State.BlindState.CurrentLevelIndex)
	for _, blindLevel := range table.State.BlindState.LevelStates {
		assert.NotEqual(t, -1, blindLevel.LevelEndAt)
	}
	assert.Equal(t, model.TableStateStatus_TableGameMatchOpen, table.State.Status)
	assert.Greater(t, table.State.GameCount, 0)
	assert.NotZero(t, table.State.GameState)
}

func TestPlayerJoin_BuyIn(t *testing.T) {
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	tableSetting := NewDefaultTableSetting()
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	joinPlayers := []model.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 1000},
		{PlayerID: "Chuck", RedeemChips: 1000},
		{PlayerID: "Fred", RedeemChips: 1000},
	}

	for _, joinPlayer := range joinPlayers {
		table, err = tableEngine.PlayerJoin(table, joinPlayer)
		assert.Nil(t, err)
	}

	assert.Equal(t, len(joinPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.PlayerSeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, len(joinPlayers), seatTakenCount)
}

func TestPlayerJoin_ReBuy(t *testing.T) {
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	initialPlayers := []model.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 0},
		{PlayerID: "Chuck", RedeemChips: 1000},
		{PlayerID: "Fred", RedeemChips: 1000},
	}
	tableSetting := NewDefaultTableSetting(initialPlayers...)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	reBuyPlayer := initialPlayers[0]
	reBuyPlayer.RedeemChips = 2000
	table, err = tableEngine.PlayerJoin(table, reBuyPlayer)
	assert.Nil(t, err)

	assert.Equal(t, len(initialPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.PlayerSeatMap {
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
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	initialPlayers := []model.JoinPlayer{
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
	table, err = tableEngine.PlayerJoin(table, addonPlayer)
	assert.Nil(t, err)

	assert.Equal(t, len(initialPlayers), len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.PlayerSeatMap {
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
	gameEngine := NewGameEngine()
	tableEngine := NewTableEngine(gameEngine)
	initialPlayers := []model.JoinPlayer{
		{PlayerID: "Jeffrey", RedeemChips: 1000},
		{PlayerID: "Chuck", RedeemChips: 0},
		{PlayerID: "Fred", RedeemChips: 1000},
	}
	tableSetting := NewDefaultTableSetting(initialPlayers...)
	table, err := tableEngine.CreateTable(tableSetting)
	assert.Nil(t, err)

	leavePlayerIDs := []string{initialPlayers[1].PlayerID}
	expectedPlayerCount := len(initialPlayers) - len(leavePlayerIDs)

	table = tableEngine.PlayersLeave(table, leavePlayerIDs)

	assert.Equal(t, expectedPlayerCount, len(table.State.PlayerStates))
	seatTakenCount := 0
	for _, playerIdx := range table.State.PlayerSeatMap {
		if playerIdx != -1 {
			seatTakenCount++
		}
	}
	assert.Equal(t, expectedPlayerCount, seatTakenCount)
}

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

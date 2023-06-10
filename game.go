package pokertable

import (
	"github.com/weedbox/pokerface"
)

func NewGame(table *Table) pokerface.Game {
	rule := table.Meta.CompetitionMeta.Rule
	blind := table.State.BlindState.LevelStates[table.State.BlindState.CurrentLevelIndex]
	dealerBlindTimes := table.Meta.CompetitionMeta.Blind.DealerBlindTimes

	// create game options
	opts := pokerface.NewStardardGameOptions()
	opts.Deck = pokerface.NewStandardDeckCards()

	if rule == CompetitionRule_ShortDeck {
		opts = pokerface.NewShortDeckGameOptions()
		opts.Deck = pokerface.NewShortDeckCards()
	} else if rule == CompetitionRule_Omaha {
		opts.HoleCardsCount = 4
		opts.RequiredHoleCardsCount = 2
	}

	// preparing blind
	opts.Ante = blind.AnteChips
	opts.Blind = pokerface.BlindSetting{
		Dealer: blind.AnteChips * (int64(dealerBlindTimes) - 1),
		SB:     blind.SBChips,
		BB:     blind.BBChips,
	}

	// preparing players
	playerSettings := make([]*pokerface.PlayerSetting, 0)
	for _, playerIdx := range table.State.GamePlayerIndexes {
		player := table.State.PlayerStates[playerIdx]
		playerSettings = append(playerSettings, &pokerface.PlayerSetting{
			Bankroll:  player.Bankroll,
			Positions: player.Positions,
		})
	}
	opts.Players = playerSettings

	// create game
	return pokerface.NewPokerFace().NewGame(opts)
}

func GameEventName(event pokerface.GameEvent) string {
	return pokerface.GameEventSymbols[event]
}

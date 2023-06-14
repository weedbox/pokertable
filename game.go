package pokertable

import (
	"github.com/weedbox/pokerface"
)

func NewGame(table *Table) pokerface.Game {
	rule := table.Meta.CompetitionMeta.Rule
	blind := table.State.BlindState.LevelStates[table.State.BlindState.CurrentLevelIndex].BlindLevel
	DealerBlindTime := table.Meta.CompetitionMeta.Blind.DealerBlindTime

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
	dealer := int64(0)
	if DealerBlindTime > 0 {
		dealer = blind.Ante * (int64(DealerBlindTime) - 1)
	}

	opts.Ante = blind.Ante
	opts.Blind = pokerface.BlindSetting{
		Dealer: dealer,
		SB:     blind.SB,
		BB:     blind.BB,
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

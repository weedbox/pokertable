package pokertable

import (
	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/syncsaga"
)

func (te *tableEngine) updateGameState(tableGame *TableGame) {
	handlers := map[string]func(*TableGame){
		gameEvent(pokerface.GameEvent_ReadyRequested):  te.onReadyRequested,
		gameEvent(pokerface.GameEvent_AnteRequested):   te.onAnteRequested,
		gameEvent(pokerface.GameEvent_BlindsRequested): te.onBlindsRequested,
		gameEvent(pokerface.GameEvent_RoundClosed):     te.onRoundClosed,
		gameEvent(pokerface.GameEvent_GameClosed):      te.onGameClosed,
	}
	handler, exist := handlers[tableGame.Game.GetState().Status.CurrentEvent]
	if !exist {
		return
	}
	handler(tableGame)
}

func (te *tableEngine) onReadyRequested(tableGame *TableGame) {
	gs := tableGame.Game.GetState()

	// Preparing ready group to wait for all player ready
	tableGame.GameReadyGroup.Stop()
	tableGame.GameReadyGroup.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		err := tableGame.Game.ReadyForAll()
		if err != nil {
			te.emitErrorEvent("ReadyForAll", "", err, tableGame.Table)
			return
		}

		// reset AllowedActions
		for _, p := range gs.Players {
			if funk.Contains(p.AllowedActions, Action_Ready) {
				p.AllowedActions = funk.Filter(p.AllowedActions, func(action string) bool {
					return action != Action_Ready
				}).([]string)
			}
		}

		te.updateGameState(tableGame)
		te.emitEvent("ReadyForAll", "", tableGame.Table)
	})

	tableGame.GameReadyGroup.ResetParticipants()
	for _, p := range gs.Players {
		tableGame.GameReadyGroup.Add(int64(p.Idx), false)

		// Allow "ready" action
		p.AllowAction(Action_Ready)
	}

	tableGame.GameReadyGroup.Start()
}

func (te *tableEngine) onAnteRequested(tableGame *TableGame) {
	gs := tableGame.Game.GetState()

	if gs.Meta.Ante == 0 {
		return
	}

	// Preparing ready group to wait for ante paid from all player
	tableGame.GameReadyGroup.Stop()
	tableGame.GameReadyGroup.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		err := tableGame.Game.PayAnte()
		if err != nil {
			te.emitErrorEvent("PayAnte", "", err, tableGame.Table)
			return
		}

		// reset AllowedActions
		for _, p := range gs.Players {
			if funk.Contains(p.AllowedActions, Action_Pay) {
				p.AllowedActions = funk.Filter(p.AllowedActions, func(action string) bool {
					return action != Action_Pay
				}).([]string)
			}
		}

		te.updateGameState(tableGame)
		te.emitEvent("PayAnte", "", tableGame.Table)
	})

	tableGame.GameReadyGroup.ResetParticipants()
	for _, p := range gs.Players {
		tableGame.GameReadyGroup.Add(int64(p.Idx), false)

		// Allow "pay" action
		p.AllowAction(Action_Pay)
	}

	tableGame.GameReadyGroup.Start()
}

func (te *tableEngine) onBlindsRequested(tableGame *TableGame) {
	gs := tableGame.Game.GetState()

	// Preparing ready group to wait for blinds
	tableGame.GameReadyGroup.Stop()
	tableGame.GameReadyGroup.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		err := tableGame.Game.PayBlinds()
		if err != nil {
			te.emitErrorEvent("PayBlinds", "", err, tableGame.Table)
			return
		}

		// reset AllowedActions
		for _, p := range gs.Players {
			if funk.Contains(p.AllowedActions, Action_Pay) {
				p.AllowedActions = funk.Filter(p.AllowedActions, func(action string) bool {
					return action != Action_Pay
				}).([]string)
			}
		}

		te.updateGameState(tableGame)
		te.emitEvent("PayBlinds", "", tableGame.Table)
	})

	tableGame.GameReadyGroup.ResetParticipants()
	for _, p := range gs.Players {
		// Allow "pay" action
		if gs.Meta.Blind.BB > 0 && gs.HasPosition(p.Idx, Position_BB) {
			tableGame.GameReadyGroup.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		} else if gs.Meta.Blind.SB > 0 && gs.HasPosition(p.Idx, Position_SB) {
			tableGame.GameReadyGroup.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		} else if gs.Meta.Blind.Dealer > 0 && gs.HasPosition(p.Idx, Position_Dealer) {
			tableGame.GameReadyGroup.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		}
	}

	tableGame.GameReadyGroup.Start()
}

func (te *tableEngine) onRoundClosed(tableGame *TableGame) {
	if err := tableGame.Game.Next(); err != nil {
		return
	}
	te.emitEvent("Auto Next Round", "", tableGame.Table)
	te.updateGameState(tableGame)
}

func (te *tableEngine) onGameClosed(tableGame *TableGame) {
	te.settleTableGame(tableGame)
}

func newGame(table *Table) pokerface.Game {
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

func gameEvent(event pokerface.GameEvent) string {
	return pokerface.GameEventSymbols[event]
}

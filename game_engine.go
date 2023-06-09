package pokertable

import (
	"github.com/weedbox/pokerface"
)

type GameEngine struct {
	game pokerface.Game
}

func NewGameEngine() *GameEngine {
	return &GameEngine{}
}

func (ge *GameEngine) GameState() pokerface.GameState {
	return *ge.game.GetState()
}

func (ge *GameEngine) Start(setting GameEngineSetting) error {
	// creating pokerface game
	pf := pokerface.NewPokerFace()
	opts := pokerface.NewStardardGameOptions()

	// preparing deck
	opts.Deck = pokerface.NewStandardDeckCards()

	// TODO: implement Rule_Omaha
	if setting.Rule == CompetitionRule_ShortDeck {
		opts = pokerface.NewShortDeckGameOptions()
		opts.Deck = pokerface.NewShortDeckCards()
	}

	// preparing blind
	opts.Ante = setting.Ante
	opts.Blind = setting.Blind

	// preparing players
	opts.Players = setting.Players

	// set to engine
	ge.game = pf.NewGame(opts)

	// start game
	err := ge.game.Start()

	return err
}

func (ge *GameEngine) NextRound() error {
	return ge.game.Next()
}

func (ge *GameEngine) AllPlayersReady() error {
	return ge.game.ReadyForAll()
}

func (ge *GameEngine) PlayerReady(playerIdx int) error {
	return ge.game.Ready(playerIdx)
}

func (ge *GameEngine) PayAnte() error {
	return ge.game.PayAnte()
}

func (ge *GameEngine) PaySB() error {
	return ge.Pay(ge.GameState().Meta.Blind.SB)
}

func (ge *GameEngine) PayBB() error {
	return ge.Pay(ge.GameState().Meta.Blind.BB)
}

func (ge *GameEngine) Pay(chips int64) error {
	return ge.game.Pay(chips)
}

func (ge *GameEngine) Bet(chips int64) error {
	return ge.game.Bet(chips)
}

func (ge *GameEngine) Raise(chipLevel int64) error {
	return ge.game.Raise(chipLevel)
}

func (ge *GameEngine) Call() error {
	return ge.game.Call()
}

func (ge *GameEngine) Allin() error {
	return ge.game.Allin()
}

func (ge *GameEngine) Check() error {
	return ge.game.Check()
}

func (ge *GameEngine) Fold() error {
	return ge.game.Fold()
}

func (ge *GameEngine) GameEventName(event pokerface.GameEvent) string {
	return pokerface.GameEventSymbols[event]
}

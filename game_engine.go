package pokertable

import (
	"github.com/weedbox/pokerface"
)

type GameEngine interface {
	// Getters
	GameState() pokerface.GameState

	// Core Actions
	Start(setting GameEngineSetting) (pokerface.GameState, error)
	NextRound() (pokerface.GameState, error)

	// Player Actions
	PlayerReady(playerIdx int) (pokerface.GameState, error)
	AllPlayersReady() (pokerface.GameState, error)
	PayAnte() (pokerface.GameState, error)
	PaySB() (pokerface.GameState, error)
	PayBB() (pokerface.GameState, error)
	Pay(chips int64) (pokerface.GameState, error)
	Bet(chips int64) (pokerface.GameState, error)
	Raise(chipLevel int64) (pokerface.GameState, error)
	Call() (pokerface.GameState, error)
	Allin() (pokerface.GameState, error)
	Check() (pokerface.GameState, error)
	Fold() (pokerface.GameState, error)
}

type gameEngine struct {
	game pokerface.Game
}

func NewGameEngine() GameEngine {
	return &gameEngine{}
}

func (engine *gameEngine) GameState() pokerface.GameState {
	return *engine.game.GetState()
}

func (engine *gameEngine) Start(setting GameEngineSetting) (pokerface.GameState, error) {
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
	engine.game = pf.NewGame(opts)

	// start game
	err := engine.game.Start()

	return engine.GameState(), err
}

func (engine *gameEngine) NextRound() (pokerface.GameState, error) {
	err := engine.game.Next()
	return engine.GameState(), err
}

func (engine *gameEngine) AllPlayersReady() (pokerface.GameState, error) {
	err := engine.game.ReadyForAll()
	return engine.GameState(), err
}

func (engine *gameEngine) PlayerReady(playerIdx int) (pokerface.GameState, error) {
	err := engine.game.Ready(playerIdx)
	return engine.GameState(), err
}

func (engine *gameEngine) PayAnte() (pokerface.GameState, error) {
	err := engine.game.PayAnte()
	if err != nil {
		return engine.GameState(), err
	}
	return engine.GameState(), nil
}

func (engine *gameEngine) PaySB() (pokerface.GameState, error) {
	err := engine.game.Pay(engine.GameState().Meta.Blind.SB)
	if err != nil {
		return engine.GameState(), err
	}
	return engine.GameState(), nil
}

func (engine *gameEngine) PayBB() (pokerface.GameState, error) {
	err := engine.game.Pay(engine.GameState().Meta.Blind.BB)
	if err != nil {
		return engine.GameState(), err
	}
	return engine.GameState(), nil
}

func (engine *gameEngine) Pay(chips int64) (pokerface.GameState, error) {
	err := engine.game.Pay(chips)
	return engine.GameState(), err
}

func (engine *gameEngine) Bet(chips int64) (pokerface.GameState, error) {
	err := engine.game.Bet(chips)
	return engine.GameState(), err
}

func (engine *gameEngine) Raise(chipLevel int64) (pokerface.GameState, error) {
	err := engine.game.Raise(chipLevel)
	return engine.GameState(), err
}

func (engine *gameEngine) Call() (pokerface.GameState, error) {
	err := engine.game.Call()
	return engine.GameState(), err
}

func (engine *gameEngine) Allin() (pokerface.GameState, error) {
	err := engine.game.Allin()
	return engine.GameState(), err
}

func (engine *gameEngine) Check() (pokerface.GameState, error) {
	err := engine.game.Check()
	return engine.GameState(), err
}

func (engine *gameEngine) Fold() (pokerface.GameState, error) {
	err := engine.game.Fold()
	return engine.GameState(), err
}

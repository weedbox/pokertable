package pokertable

import (
	"fmt"

	"github.com/weedbox/pokerface"
)

type GameEngine interface {
	GameState() pokerface.GameState
	Start(GameEngineSetting) (pokerface.GameState, error)
	PlayerReady(int) (pokerface.GameState, error)
	AllPlayersReady() (pokerface.GameState, error)
	PayAnte() (pokerface.GameState, error)
	PaySB_BB() (pokerface.GameState, error)
	PlayerWager(string, int64) (pokerface.GameState, error)
	NextRound() (pokerface.GameState, error)
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
		fmt.Printf("[gameEngine#PayAnte] players auto pay ante error: %+v\n", err)
		return engine.GameState(), err
	}
	fmt.Println("[gameEngine#PayAnte] dealer receive ante for all players.")
	return engine.GameState(), nil
}

func (engine *gameEngine) PaySB_BB() (pokerface.GameState, error) {
	blind := engine.GameState().Meta.Blind
	for _, p := range engine.game.GetPlayers() {
		if p.SeatIndex() == 1 {
			// Small Blind
			if err := p.Pay(blind.SB); err != nil {
				fmt.Printf("[gameEngine#PaySB_BB] player auto pay small blind(%d) error: %+v\n", blind.SB, err)
				return engine.GameState(), err
			}
			fmt.Printf("[gameEngine#PaySB_BB] dealer receive small blind(%d).\n", blind.SB)
		} else if p.SeatIndex() == 2 {
			// Big Blind
			if err := p.Pay(blind.BB); err != nil {
				fmt.Printf("[gameEngine#PaySB_BB] player auto pay big blind(%d) error: %+v\n", blind.BB, err)
				return engine.GameState(), err
			}
			fmt.Printf("[gameEngine#PaySB_BB] dealer receive big blind(%d).\n", blind.BB)
		}
	}
	return engine.GameState(), nil
}

func (engine *gameEngine) PlayerWager(action string, chips int64) (pokerface.GameState, error) {
	var err error
	switch action {
	case WagerAction_Fold:
		err = engine.game.Fold()
	case WagerAction_Check:
		err = engine.game.Check()
	case WagerAction_Call:
		err = engine.game.Call()
	case WagerAction_AllIn:
		err = engine.game.Allin()
	case WagerAction_Bet:
		err = engine.game.Bet(chips)
	case WagerAction_Raise:
		err = engine.game.Raise(chips)
	}
	return engine.GameState(), err
}

func (engine *gameEngine) NextRound() (pokerface.GameState, error) {
	err := engine.game.Next()
	return engine.GameState(), err
}

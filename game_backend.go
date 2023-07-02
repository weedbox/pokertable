package pokertable

import (
	"encoding/json"

	"github.com/weedbox/pokerface"
)

type GameBackend struct {
	engine pokerface.PokerFace
}

func NewGameBackend() *GameBackend {
	return &GameBackend{
		engine: pokerface.NewPokerFace(),
	}
}

func cloneGameState(gs *pokerface.GameState) *pokerface.GameState {
	// Note: we must clone a new structure for preventing original data of game engine is modified outside.
	data, err := json.Marshal(gs)
	if err != nil {
		return nil
	}

	var state pokerface.GameState
	err = json.Unmarshal([]byte(data), &state)
	if err != nil {
		return nil
	}

	return &state
}

func (gb *GameBackend) getState(g pokerface.Game) *pokerface.GameState {
	return cloneGameState(g.GetState())
}

func (gb *GameBackend) CreateGame(opts *pokerface.GameOptions) (*pokerface.GameState, error) {
	// Initializing game
	g := gb.engine.NewGame(opts)
	err := g.Start()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) ReadyForAll(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.ReadyForAll()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) PayAnte(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayAnte()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) PayBlinds(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayBlinds()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Next(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Next()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Pay(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pay(chips)
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Fold(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Fold()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Check(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Check()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Call(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Call()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Allin(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Allin()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Bet(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Bet(chips)
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Raise(gs *pokerface.GameState, chipLevel int64) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Raise(chipLevel)
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

func (gb *GameBackend) Pass(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := gb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pass()
	if err != nil {
		return nil, err
	}

	return gb.getState(g), nil
}

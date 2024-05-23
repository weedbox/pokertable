package pokertable

import (
	"encoding/json"

	"github.com/weedbox/pokerface"
)

type NativeGameBackend struct {
	engine pokerface.PokerFace
}

func NewNativeGameBackend() *NativeGameBackend {
	return &NativeGameBackend{
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

func (ngb *NativeGameBackend) getState(g pokerface.Game) *pokerface.GameState {
	return cloneGameState(g.GetState())
}

func (ngb *NativeGameBackend) CreateGame(opts *pokerface.GameOptions) (*pokerface.GameState, error) {
	g := ngb.engine.NewGame(opts)
	err := g.Start()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) ReadyForAll(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.ReadyForAll()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) PayAnte(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayAnte()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) PayBlinds(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayBlinds()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Next(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Next()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Pay(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pay(chips)
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Fold(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Fold()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Check(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Check()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Call(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Call()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Allin(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Allin()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Bet(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Bet(chips)
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Raise(gs *pokerface.GameState, chipLevel int64) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Raise(chipLevel)
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

func (ngb *NativeGameBackend) Pass(gs *pokerface.GameState) (*pokerface.GameState, error) {
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pass()
	if err != nil {
		return nil, err
	}
	return ngb.getState(g), nil
}

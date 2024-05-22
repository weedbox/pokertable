package pokertable

import (
	"encoding/json"
	"fmt"

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
	fmt.Println("[DEBUG-LOG#CreateGame] in game id: X")
	// Initializing game
	g := ngb.engine.NewGame(opts)
	err := g.Start()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Println("[DEBUG-LOG#CreateGame] out game id:", newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) ReadyForAll(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#ReadyForAll] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.ReadyForAll()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#ReadyForAll] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) PayAnte(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#PayAnte] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayAnte()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#PayAnte] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) PayBlinds(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#PayBlinds] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.PayBlinds()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#PayBlinds] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Next(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Next] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Next()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Next] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Pay(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Pay] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pay(chips)
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Pay] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Fold(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Fold] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Fold()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Fold] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Check(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Check] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Check()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Check] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Call(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Call] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Call()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Call] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Allin(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Allin] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Allin()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Allin] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Bet(gs *pokerface.GameState, chips int64) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Bet] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Bet(chips)
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Bet] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Raise(gs *pokerface.GameState, chipLevel int64) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Raise] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Raise(chipLevel)
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Raise] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

func (ngb *NativeGameBackend) Pass(gs *pokerface.GameState) (*pokerface.GameState, error) {
	fmt.Println("[DEBUG-LOG#Pass] in game id:", gs.GameID)
	g := ngb.engine.NewGameFromState(cloneGameState(gs))
	err := g.Pass()
	if err != nil {
		return nil, err
	}

	newGS := ngb.getState(g)
	fmt.Printf("[DEBUG-LOG#Pass] [%+v] out game id: %s\n", newGS.GameID == gs.GameID, newGS.GameID)
	return newGS, nil
}

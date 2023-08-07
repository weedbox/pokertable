package pokertable

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/thoas/go-funk"
	"github.com/weedbox/pokerface"
	"github.com/weedbox/syncsaga"
)

var (
	ErrGamePlayerNotFound      = errors.New("game: player not found")
	ErrGameInvalidAction       = errors.New("game: invalid action")
	ErrGameUnknownEvent        = errors.New("game: unknown event")
	ErrGameUnknownEventHandler = errors.New("game: unknown event handler")
)

type Game interface {
	// Events
	OnGameStateUpdated(func(*pokerface.GameState))
	OnGameErrorUpdated(func(*pokerface.GameState, error))

	// Others
	GetGameState() *pokerface.GameState
	Start() error
	Next() error

	// Group Actions
	ReadyForAll() error
	PayAnte() error
	PayBlinds() error

	// Single Actions
	Ready(playerIdx int) error
	Pay(playerIdx int, chips int64) error
	Pass(playerIdx int) error
	Fold(playerIdx int) error
	Check(playerIdx int) error
	Call(playerIdx int) error
	Allin(playerIdx int) error
	Bet(playerIdx int, chips int64) error
	Raise(playerIdx int, chipLevel int64) error
}

type game struct {
	backend            GameBackend
	gs                 *pokerface.GameState
	opts               *pokerface.GameOptions
	rg                 *syncsaga.ReadyGroup
	mu                 sync.RWMutex
	isClosed           bool
	incomingStates     chan *pokerface.GameState
	onGameStateUpdated func(*pokerface.GameState)
	onGameErrorUpdated func(*pokerface.GameState, error)
}

func NewGame(backend GameBackend, opts *pokerface.GameOptions) *game {
	rg := syncsaga.NewReadyGroup(
		syncsaga.WithTimeout(1, func(rg *syncsaga.ReadyGroup) {
			// Auto Ready By Default
			states := rg.GetParticipantStates()
			for gamePlayerIdx, isReady := range states {
				if !isReady {
					rg.Ready(gamePlayerIdx)
				}
			}
		}),
	)
	return &game{
		backend:            backend,
		opts:               opts,
		rg:                 rg,
		incomingStates:     make(chan *pokerface.GameState, 1024),
		onGameStateUpdated: func(gs *pokerface.GameState) {},
		onGameErrorUpdated: func(gs *pokerface.GameState, err error) {},
	}
}

func (g *game) OnGameStateUpdated(fn func(*pokerface.GameState)) {
	g.onGameStateUpdated = fn
}

func (g *game) OnGameErrorUpdated(fn func(*pokerface.GameState, error)) {
	g.onGameErrorUpdated = fn
}

func (g *game) GetGameState() *pokerface.GameState {
	return g.gs
}

func (g *game) Start() error {
	g.runGameStateUpdater()

	gs, err := g.backend.CreateGame(g.opts)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Next() error {
	gs, err := g.backend.Next(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) ReadyForAll() error {
	gs, err := g.backend.ReadyForAll(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) PayAnte() error {
	gs, err := g.backend.PayAnte(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) PayBlinds() error {
	gs, err := g.backend.PayBlinds(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Ready(playerIdx int) error {
	if err := g.validateActionMove(playerIdx, Action_Ready); err != nil {
		return err
	}

	g.rg.Ready(int64(playerIdx))
	return nil
}

func (g *game) Pay(playerIdx int, chips int64) error {
	if err := g.validateActionMove(playerIdx, Action_Pay); err != nil {
		return err
	}

	event, ok := pokerface.GameEventBySymbol[g.gs.Status.CurrentEvent]
	if !ok {
		return ErrGameUnknownEvent
	}

	// For blinds
	switch event {
	case pokerface.GameEvent_AnteRequested:
		fallthrough
	case pokerface.GameEvent_BlindsRequested:
		g.rg.Ready(int64(playerIdx))
		return nil
	}

	gs, err := g.backend.Pay(g.gs, chips)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Pass(playerIdx int) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Pass(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Fold(playerIdx int) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Fold(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Check(playerIdx int) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Check(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Call(playerIdx int) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Call(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Allin(playerIdx int) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Allin(g.gs)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Bet(playerIdx int, chips int64) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Bet(g.gs, chips)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) Raise(playerIdx int, chipLevel int64) error {
	if err := g.validatePlayMove(playerIdx); err != nil {
		return err
	}

	gs, err := g.backend.Raise(g.gs, chipLevel)
	if err != nil {
		return err
	}

	g.updateGameState(gs)
	return nil
}

func (g *game) validatePlayMove(playerIdx int) error {
	if p := g.gs.GetPlayer(playerIdx); p == nil {
		return ErrGamePlayerNotFound
	}

	if g.gs.Status.CurrentPlayer != playerIdx {
		// fmt.Printf("[DEBUG#game] not player's turn. CurrentPlayer: %d, playerIdx: %d\n", g.gs.Status.CurrentPlayer, playerIdx)
		return ErrGameInvalidAction
	}

	return nil
}

func (g *game) validateActionMove(playerIdx int, action string) error {
	if p := g.gs.GetPlayer(playerIdx); p == nil {
		return ErrGamePlayerNotFound
	}

	if !g.gs.HasAction(playerIdx, action) {
		// fmt.Printf("[DEBUG#game] player (%d) has no action (%s)\n", playerIdx, action)
		return ErrGameInvalidAction
	}

	if g.rg == nil {
		// fmt.Println("[DEBUG#game] ready group is nil")
		return ErrGameInvalidAction
	}

	return nil
}

func (g *game) runGameStateUpdater() {
	go func() {
		for state := range g.incomingStates {
			g.handleGameState(state)
		}
	}()
}

func (g *game) cloneState(gs *pokerface.GameState) *pokerface.GameState {
	// clone table state
	data, err := json.Marshal(gs)
	if err != nil {
		return nil
	}

	var state pokerface.GameState
	json.Unmarshal(data, &state)

	return &state
}

func (g *game) updateGameState(gs *pokerface.GameState) {
	g.mu.Lock()
	defer g.mu.Unlock()

	state := g.cloneState(gs)
	g.gs = state

	if g.isClosed {
		return
	}

	g.incomingStates <- state
}

func (g *game) handleGameState(gs *pokerface.GameState) {
	event, ok := pokerface.GameEventBySymbol[gs.Status.CurrentEvent]
	if !ok {
		g.onGameErrorUpdated(gs, ErrGameUnknownEvent)
		return
	}

	handlers := map[pokerface.GameEvent]func(*pokerface.GameState){
		pokerface.GameEvent_ReadyRequested:  g.onReadyRequested,
		pokerface.GameEvent_AnteRequested:   g.onAnteRequested,
		pokerface.GameEvent_BlindsRequested: g.onBlindsRequested,
		pokerface.GameEvent_RoundClosed:     g.onRoundClosed,
		pokerface.GameEvent_GameClosed:      g.onGameClosed,
	}
	if handler, exist := handlers[event]; exist {
		handler(gs)
	}
	g.onGameStateUpdated(gs)
}

func (g *game) onReadyRequested(gs *pokerface.GameState) {
	// Preparing ready group to wait for all player ready
	g.rg.Stop()
	g.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		if err := g.ReadyForAll(); err != nil {
			g.onGameErrorUpdated(gs, err)
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
	})

	g.rg.ResetParticipants()
	for _, p := range gs.Players {
		g.rg.Add(int64(p.Idx), false)

		// Allow "ready" action
		p.AllowAction(Action_Ready)
	}

	g.rg.Start()
}

func (g *game) onAnteRequested(gs *pokerface.GameState) {
	if gs.Meta.Ante == 0 {
		return
	}

	// Preparing ready group to wait for ante paid from all player
	g.rg.Stop()
	g.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		if err := g.PayAnte(); err != nil {
			g.onGameErrorUpdated(gs, err)
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
	})

	g.rg.ResetParticipants()
	for _, p := range gs.Players {
		g.rg.Add(int64(p.Idx), false)

		// Allow "pay" action
		p.AllowAction(Action_Pay)
	}

	g.rg.Start()
}

func (g *game) onBlindsRequested(gs *pokerface.GameState) {
	// Preparing ready group to wait for blinds
	g.rg.Stop()
	g.rg.OnCompleted(func(rg *syncsaga.ReadyGroup) {
		if err := g.PayBlinds(); err != nil {
			g.onGameErrorUpdated(gs, err)
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
	})

	g.rg.ResetParticipants()
	for _, p := range gs.Players {
		// Allow "pay" action
		if gs.Meta.Blind.BB > 0 && gs.HasPosition(p.Idx, Position_BB) {
			g.rg.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		} else if gs.Meta.Blind.SB > 0 && gs.HasPosition(p.Idx, Position_SB) {
			g.rg.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		} else if gs.Meta.Blind.Dealer > 0 && gs.HasPosition(p.Idx, Position_Dealer) {
			g.rg.Add(int64(p.Idx), false)
			p.AllowAction(Action_Pay)
		}
	}

	g.rg.Start()
}

func (g *game) onRoundClosed(gs *pokerface.GameState) {
	// Next round automatically
	gs, err := g.backend.Next(gs)
	if err != nil {
		g.onGameErrorUpdated(gs, err)
		return
	}

	g.updateGameState(gs)
}

func (g *game) onGameClosed(gs *pokerface.GameState) {
	if g.isClosed {
		return
	}

	g.isClosed = true
	close(g.incomingStates)
}

package actor

import (
	"math/rand"
	"time"

	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokertable"
	"github.com/weedbox/timebank"
)

type TableAutoJoinActionRequestFunc func(competitionID, tableID, playerID string)

type ActionProbability struct {
	Action string
	Weight float64
}

var (
	actionProbabilities = []ActionProbability{
		{Action: "check", Weight: 0.1},
		{Action: "call", Weight: 0.3},
		{Action: "fold", Weight: 0.15},
		{Action: "allin", Weight: 0.05},
		{Action: "raise", Weight: 0.3},
		{Action: "bet", Weight: 0.1},
	}
)

type botRunner struct {
	actor                          Actor
	actions                        Actions
	playerID                       string
	isHumanized                    bool
	curGameID                      string
	lastGameStateTime              int64
	timebank                       *timebank.TimeBank
	tableInfo                      *pokertable.Table
	onTableGameWagerActionUpdated  TableGameWagerActionUpdatedFunc
	onTableAutoJoinActionRequested TableAutoJoinActionRequestFunc
}

func NewBotRunner(playerID string) *botRunner {
	return &botRunner{
		playerID:                       playerID,
		timebank:                       timebank.NewTimeBank(),
		onTableGameWagerActionUpdated:  func(string, string, int, string, string, int64) {},
		onTableAutoJoinActionRequested: func(string, string, string) {},
	}
}

func (br *botRunner) SetActor(a Actor) {
	br.actor = a
	br.actions = NewActions(a, br.playerID)
}

func (br *botRunner) Humanized(enabled bool) {
	br.isHumanized = enabled
}

func (br *botRunner) OnTableGameWagerActionUpdated(fn TableGameWagerActionUpdatedFunc) error {
	br.onTableGameWagerActionUpdated = fn
	return nil
}

func (br *botRunner) OnTableAutoJoinActionRequested(fn TableAutoJoinActionRequestFunc) error {
	br.onTableAutoJoinActionRequested = fn
	return nil
}

func (br *botRunner) UpdateTableState(table *pokertable.Table) error {

	gs := table.State.GameState
	//oldState := br.tableInfo.State
	br.tableInfo = table

	// Check if you have been eliminated
	isEliminated := true
	shouldAutoJoin := false
	for _, ps := range table.State.PlayerStates {
		if ps.PlayerID == br.playerID {
			isEliminated = false
			if !ps.IsIn {
				shouldAutoJoin = true
			}
			break
		}
	}

	if isEliminated {
		// fmt.Printf("[DEBUG#botRunner#UpdateTableState] [1] No reaction required since bot (%s), table (%s) is eliminated.\n", br.playerID, table.ID)
		return nil
	}

	// request auto join table
	if shouldAutoJoin {
		return br.timebank.NewTask(time.Duration(100)*time.Millisecond, func(isCancelled bool) {

			if isCancelled {
				return
			}

			br.onTableAutoJoinActionRequested(table.Meta.CompetitionID, table.ID, br.playerID)
		})
	}

	// The state remains unchanged or is outdated
	if gs != nil {

		// New game
		if gs.GameID != br.curGameID {
			br.curGameID = gs.GameID
		} else if br.lastGameStateTime >= gs.UpdatedAt {
			// Ignore if game state is too old
			//fmt.Println(br.playerID, table.ID)
			// fmt.Printf("[DEBUG#botRunner#UpdateTableState] [2] No reaction required since lastGameStateTime >= currentGameStateTime. Bot PlayerID: %s, TableID: %s\n", br.playerID, table.ID)
			return nil
		}

		br.lastGameStateTime = gs.UpdatedAt
	}

	// game move is allowed when the game is playing
	if table.State.Status != pokertable.TableStateStatus_TableGamePlaying {
		return nil
	}

	// Getting player index in game
	gamePlayerIdx := table.GamePlayerIndex(br.playerID)

	// Somehow, this player is not in the game.
	// It probably has no chips already or just sat down and have not participated in the game yet
	if gamePlayerIdx == -1 {
		// fmt.Printf("[DEBUG#botRunner#UpdateTableState] [3] No reaction required since bot (%s) gamePlayerIdx is -1 at table (%s) when table status is %s.\n", br.playerID, table.ID, pokertable.TableStateStatus_TableGamePlaying)
		return nil
	}

	//fmt.Printf("Bot (player_id=%s, gameIdx=%d, event=%s)\n", br.playerID, gamePlayerIdx, gs.Status.CurrentEvent)

	// game is running so we have to check actions allowed
	player := gs.GetPlayer(gamePlayerIdx)
	if player == nil {
		// fmt.Printf("[DEBUG#botRunner#UpdateTableState] [4] No reaction required since gs GetPlayer(Bot) Failed. PlayerID: %s, GameIdx: %d, Event: %s\n", br.playerID, gamePlayerIdx, gs.Status.CurrentEvent)
		return nil
	}

	if len(player.AllowedActions) > 0 {
		//fmt.Println(br.playerID, player.AllowedActions)
		err := br.requestMove(table.State.GameState, gamePlayerIdx)
		if err != nil {
			// fmt.Printf("[DEBUG#botRunner#UpdateTableState] [5] Bot requestMove Failed (player_id=%s, gameIdx=%d, event=%s). Error: %+v\n", br.playerID, gamePlayerIdx, gs.Status.CurrentEvent, err)
			return err
		}
	}

	return nil
}

func (br *botRunner) requestMove(gs *pokerface.GameState, playerIdx int) error {

	//fmt.Println(br.tableInfo.State.GameState.Status.Round, br.gamePlayerIdx, gs.Players[br.gamePlayerIdx].AllowedActions)
	/*
		player := gs.Players[playerIdx]
		if len(player.AllowedActions) == 1 {
			fmt.Println(br.playerID, player.AllowedActions)
		}
	*/

	// Do ready() and pay() automatically
	if gs.HasAction(playerIdx, "ready") {
		return br.actions.Ready()
	} else if gs.HasAction(playerIdx, "pass") {
		return br.actions.Pass()
	} else if gs.HasAction(playerIdx, "pay") {

		// Pay for ante and blinds
		switch gs.Status.CurrentEvent {
		case pokerface.GameEventSymbols[pokerface.GameEvent_AnteRequested]:

			// Ante
			return br.actions.Pay(gs.Meta.Ante)

		case pokerface.GameEventSymbols[pokerface.GameEvent_BlindsRequested]:

			// blinds
			if gs.HasPosition(playerIdx, "sb") {
				return br.actions.Pay(gs.Meta.Blind.SB)
			} else if gs.HasPosition(playerIdx, "bb") {
				return br.actions.Pay(gs.Meta.Blind.BB)
			}

			return br.actions.Pay(gs.Meta.Blind.Dealer)
		}
	}

	if !br.isHumanized || br.tableInfo.Meta.ActionTime == 0 {
		return br.requestAI(gs, playerIdx)
	}

	// For simulating human-like behavior, to incorporate random delays when performing actions.
	thinkingTime := rand.Intn(br.tableInfo.Meta.ActionTime)
	if thinkingTime == 0 {
		return br.requestAI(gs, playerIdx)
	}

	return br.timebank.NewTask(time.Duration(thinkingTime)*time.Second, func(isCancelled bool) {

		if isCancelled {
			return
		}

		br.requestAI(gs, playerIdx)
	})
}

func (br *botRunner) calcActionProbabilities(actions []string) map[string]float64 {

	probabilities := make(map[string]float64)
	totalWeight := 0.0
	for _, action := range actions {

		for _, p := range actionProbabilities {
			if action == p.Action {
				probabilities[action] = p.Weight
				totalWeight += p.Weight
				break
			}
		}
	}

	scaleRatio := 1.0 / totalWeight
	weightLevel := 0.0
	for action, weight := range probabilities {
		scaledWeight := weight * scaleRatio
		weightLevel += scaledWeight
		probabilities[action] = weightLevel
	}

	return probabilities
}

func (br *botRunner) calcAction(actions []string) string {

	// Select action randomly
	rand.Seed(time.Now().UnixNano())

	probabilities := br.calcActionProbabilities(actions)
	randomNum := rand.Float64()

	for action, probability := range probabilities {
		if randomNum < probability {
			return action
		}
	}

	return actions[len(actions)-1]
}

func (br *botRunner) requestAI(gs *pokerface.GameState, playerIdx int) error {

	player := gs.Players[playerIdx]

	// None of actions is allowed
	if len(player.AllowedActions) == 0 {
		return nil
	}

	action := player.AllowedActions[0]

	if len(player.AllowedActions) > 1 {
		action = br.calcAction(player.AllowedActions)
	}

	// Calculate chips
	chips := int64(0)

	/*
		// Debugging messages
		defer func() {
			if chips > 0 {
				fmt.Printf("Action %s %v %s(%d)\n", br.playerID, player.AllowedActions, action, chips)
			} else {
				fmt.Printf("Action %s %v %s\n", br.playerID, player.AllowedActions, action)
			}
		}()
	*/

	switch action {
	case "bet":

		minBet := gs.Status.MiniBet

		if player.InitialStackSize <= minBet {
			return br.actions.Bet(player.InitialStackSize)
		}

		chips = rand.Int63n(player.InitialStackSize-minBet) + minBet

		err := br.actions.Bet(chips)
		if err != nil {
			return err
		}

		br.updateWagerAction(pokertable.WagerAction_Bet, chips)
		return nil
	case "raise":

		maxChipLevel := player.InitialStackSize
		minChipLevel := gs.Status.CurrentWager + gs.Status.PreviousRaiseSize

		if maxChipLevel <= minChipLevel {
			err := br.actions.Raise(maxChipLevel)
			if err != nil {
				return err
			}

			br.updateWagerAction(pokertable.WagerAction_Raise, maxChipLevel)
			return nil
		}

		chips = rand.Int63n(maxChipLevel-minChipLevel) + minChipLevel

		err := br.actions.Raise(chips)
		if err != nil {
			return err
		}

		br.updateWagerAction(pokertable.WagerAction_Raise, chips)
		return nil
	case "call":
		// TODO: should move logic to pokertable
		wager := int64(0)
		gamePlayerIdx := br.tableInfo.FindGamePlayerIdx(br.playerID)
		if gamePlayerIdx >= 0 && br.tableInfo != nil && br.tableInfo.State.GameState != nil && gamePlayerIdx < len(br.tableInfo.State.GameState.Players) {
			wager = br.tableInfo.State.GameState.Status.CurrentWager - br.tableInfo.State.GameState.GetPlayer(gamePlayerIdx).Wager
		}

		err := br.actions.Call()
		if err != nil {
			return err
		}

		br.updateWagerAction(pokertable.WagerAction_Call, wager)
		return nil
	case "check":
		err := br.actions.Check()
		if err != nil {
			return err
		}

		br.updateWagerAction(pokertable.WagerAction_Check, 0)
		return nil
	case "allin":
		// TODO: should move logic to pokertable
		wager := int64(0)
		gamePlayerIdx := br.tableInfo.FindGamePlayerIdx(br.playerID)
		if gamePlayerIdx >= 0 && br.tableInfo != nil && br.tableInfo.State.GameState != nil && gamePlayerIdx < len(br.tableInfo.State.GameState.Players) {
			wager = br.tableInfo.State.GameState.GetPlayer(gamePlayerIdx).StackSize
		}

		err := br.actions.Allin()
		if err != nil {
			return err
		}

		br.updateWagerAction(pokertable.WagerAction_AllIn, wager)
		return nil
	}

	err := br.actions.Fold()
	if err != nil {
		return err
	}

	br.updateWagerAction(pokertable.WagerAction_Fold, 0)
	return nil
}

func (br *botRunner) updateWagerAction(action string, chips int64) {
	tableID := ""
	gameCount := 0
	round := ""
	if br.tableInfo != nil {
		tableID = br.tableInfo.ID
		gameCount = br.tableInfo.State.GameCount

		if br.tableInfo.State.GameState != nil {
			round = br.tableInfo.State.GameState.Status.Round
		}
	}
	br.onTableGameWagerActionUpdated(tableID, br.curGameID, gameCount, round, action, chips)
}

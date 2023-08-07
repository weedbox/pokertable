package actor

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/weedbox/pokerface"
	"github.com/weedbox/pokertable"
	"github.com/weedbox/timebank"
)

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
	actor             Actor
	actions           Actions
	playerID          string
	isHumanized       bool
	curGameID         string
	lastGameStateTime int64
	timebank          *timebank.TimeBank
	tableInfo         *pokertable.Table
}

func NewBotRunner(playerID string) *botRunner {
	return &botRunner{
		playerID: playerID,
		timebank: timebank.NewTimeBank(),
	}
}

func (br *botRunner) SetActor(a Actor) {
	br.actor = a
	br.actions = NewActions(a, br.playerID)
}

func (br *botRunner) Humanized(enabled bool) {
	br.isHumanized = enabled
}

func (br *botRunner) UpdateTableState(table *pokertable.Table) error {

	gs := table.State.GameState
	br.tableInfo = table

	// The state remains unchanged or is outdated
	if gs != nil {

		// New game
		if gs.GameID != br.curGameID {
			br.curGameID = gs.GameID
		}

		//fmt.Println(br.lastGameStateTime, br.tableInfo.State.GameState.UpdatedAt)
		if br.lastGameStateTime >= gs.UpdatedAt {
			//fmt.Println(br.playerID, table.ID)
			return nil
		}

		br.lastGameStateTime = gs.UpdatedAt
	}

	// Check if you have been eliminated
	isEliminated := true
	for _, ps := range table.State.PlayerStates {
		if ps.PlayerID == br.playerID {
			isEliminated = false
		}
	}

	if isEliminated {
		return nil
	}

	if table.State.Status == pokertable.TableStateStatus_TableGameStandby {
		return nil
	}

	// Update player index in game
	gamePlayerIdx := br.actor.GetTable().GetGamePlayerIndex(br.playerID)

	// Somehow, this player is not in the game.
	// It probably has no chips already or just sat down and have not participated in the game yet
	if gamePlayerIdx == -1 {
		return nil
	}

	if table.State.Status != pokertable.TableStateStatus_TableGamePlaying {
		return nil
	}

	//fmt.Printf("Bot (player_id=%s, gameIdx=%d)\n", br.playerID, br.gamePlayerIdx)

	// game is running so we have to check actions allowed
	player := gs.GetPlayer(gamePlayerIdx)
	if player == nil {
		return nil
	}

	if len(player.AllowedActions) > 0 {
		//fmt.Println(br.playerID, player.AllowedActions)
		return br.requestMove(table.State.GameState, gamePlayerIdx)
	}

	return nil
}

func (br *botRunner) requestMove(gs *pokerface.GameState, playerIdx int) error {

	//fmt.Println(br.tableInfo.State.GameState.Status.Round, br.gamePlayerIdx, gs.Players[br.gamePlayerIdx].AllowedActions)
	/*
		player := gs.Players[br.gamePlayerIdx]
		if len(player.AllowedActions) == 1 {
			fmt.Println(br.playerID, player.AllowedActions)
		}
	*/

	// Do ready() and pay() automatically
	if gs.HasAction(playerIdx, "ready") {
		fmt.Printf("[#%d][%d][%s][%s] READY\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
		return br.actions.Ready()
	} else if gs.HasAction(playerIdx, "pass") {
		fmt.Printf("[#%d][%d][%s][%s] PASS\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
		return br.actions.Pass()
	} else if gs.HasAction(playerIdx, "pay") {

		// Pay for ante and blinds
		switch gs.Status.CurrentEvent {
		case pokerface.GameEventSymbols[pokerface.GameEvent_AnteRequested]:

			// Ante
			fmt.Printf("[#%d][%d][%s][%s] PAY ANTE\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
			return br.actions.Pay(gs.Meta.Ante)

		case pokerface.GameEventSymbols[pokerface.GameEvent_BlindsRequested]:

			// blinds
			if gs.HasPosition(playerIdx, "sb") {
				fmt.Printf("[#%d][%d][%s][%s] PAY SB\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
				return br.actions.Pay(gs.Meta.Blind.SB)
			} else if gs.HasPosition(playerIdx, "bb") {
				fmt.Printf("[#%d][%d][%s][%s] PAY BB\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
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
			fmt.Printf("[#%d][%d][%s][%s] BET %d\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round, player.InitialStackSize)
			return br.actions.Bet(player.InitialStackSize)
		}

		chips = rand.Int63n(player.InitialStackSize-minBet) + minBet
		fmt.Printf("[#%d][%d][%s][%s] BET %d\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round, chips)
		return br.actions.Bet(chips)
	case "raise":

		maxChipLevel := player.InitialStackSize
		minChipLevel := gs.Status.CurrentWager + gs.Status.PreviousRaiseSize

		if maxChipLevel <= minChipLevel {
			fmt.Printf("[#%d][%d][%s][%s] RAISE %d\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round, maxChipLevel)
			return br.actions.Raise(maxChipLevel)
		}

		chips = rand.Int63n(maxChipLevel-minChipLevel) + minChipLevel
		fmt.Printf("[#%d][%d][%s][%s] RAISE %d\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round, chips)
		return br.actions.Raise(chips)
	case "call":
		fmt.Printf("[#%d][%d][%s][%s] CALL\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
		return br.actions.Call()
	case "check":
		fmt.Printf("[#%d][%d][%s][%s] CHECK\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
		return br.actions.Check()
	case "allin":
		fmt.Printf("[#%d][%d][%s][%s] ALLIN\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
		return br.actions.Allin()
	}

	fmt.Printf("[#%d][%d][%s][%s] FOLD\n", br.tableInfo.UpdateSerial, br.tableInfo.State.GameCount, br.playerID, br.tableInfo.State.GameState.Status.Round)
	return br.actions.Fold()
}

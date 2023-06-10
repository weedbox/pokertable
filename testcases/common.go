package testcases

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/weedbox/pokertable"
)

func logJSON(t *testing.T, msg string, jsonPrinter func() (*string, error)) {
	json, _ := jsonPrinter()
	t.Logf("\n===== [%s] =====\n%s\n", msg, *json)
}

func FindCurrentPlayerID(table *pokertable.Table) string {
	currGamePlayerIdx := table.State.GameState.Status.CurrentPlayer
	for gamePlayerIdx, playerIdx := range table.State.GamePlayerIndexes {
		if gamePlayerIdx == currGamePlayerIdx {
			return table.State.PlayerStates[playerIdx].PlayerID
		}
	}
	return ""
}

func PrintPlayerActionLog(table *pokertable.Table, playerID, actionLog string) {
	findPlayerIdx := func(players []*pokertable.TablePlayerState, targetPlayerID string) int {
		for idx, player := range players {
			if player.PlayerID == targetPlayerID {
				return idx
			}
		}

		return -1
	}

	positions := make([]string, 0)
	playerIdx := findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != -1 {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	fmt.Printf("[%s] %s%+v: %s\n", table.State.GameState.Status.Round, playerID, positions, actionLog)
}

func NewPlayerActionErrorLog(table *pokertable.Table, playerID, actionLog string, err error) string {
	if err == nil {
		return ""
	}

	findPlayerIdx := func(players []*pokertable.TablePlayerState, targetPlayerID string) int {
		for idx, player := range players {
			if player.PlayerID == targetPlayerID {
				return idx
			}
		}

		return -1
	}

	positions := make([]string, 0)
	playerIdx := findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != -1 {
		positions = table.State.PlayerStates[playerIdx].Positions
	}

	return fmt.Sprintf("[%s] %s%+v: %s. Error: %s\n", table.State.GameState.Status.Round, playerID, positions, actionLog, err.Error())
}

func AllGamePlayersReady(t *testing.T, tableEngine pokertable.TableEngine, table *pokertable.Table) {
	for _, playerIdx := range table.State.GamePlayerIndexes {
		player := table.State.PlayerStates[playerIdx]
		err := tableEngine.PlayerReady(table.ID, player.PlayerID)
		assert.Nil(t, err, NewPlayerActionErrorLog(table, FindCurrentPlayerID(table), "ready", err))
		PrintPlayerActionLog(table, player.PlayerID, fmt.Sprintf("ready. CurrentEvent: %s", table.State.GameState.Status.CurrentEvent.Name))
	}
}

package pokertable

import (
	"fmt"
	"time"
)

const (
	TableStateEvent_Created       = "Created"
	TableStateEvent_StatusUpdated = "StatusUpdated"
	TableStateEvent_GameUpdated   = "GameUpdated"
	TableStateEvent_GameSettled   = "GameSettled"
	TableStateEvent_PlayersLeave  = "PlayersLeave"
)

func (te *tableEngine) emitEvent(eventName string, playerID string) {
	// refresh table
	te.table.UpdateAt = time.Now().Unix()
	te.table.UpdateSerial++

	// emit event
	fmt.Printf("->[c: %s][t: %s][#%d][%d][%s] emit Event: %s\n", te.table.Meta.CompetitionID, te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName)
	te.onTableUpdated(te.table)
}

// TODO: replace err(error) with errMsg(string)
func (te *tableEngine) emitErrorEvent(eventName string, playerID string, err error) {
	fmt.Printf("->[c: %s][t: %s][#%d][%d][%s] emit ERROR Event: %s, Error: %v\n", te.table.Meta.CompetitionID, te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName, err)
	te.onTableErrorUpdated(te.table, err)
}

func (te *tableEngine) emitTableStateEvent(eventName string) {
	// emit event
	// fmt.Printf("->emit state Event: %s\n", eventName)
	te.onTableStateUpdated(eventName, te.table)
}

func (te *tableEngine) emitTablePlayerStateEvent(player *TablePlayerState) {
	// emit event
	// fmt.Printf("->emit player state Event: %s\n", player.PlayerID)
	te.onTablePlayerStateUpdated(te.table.Meta.CompetitionID, te.table.ID, player)
}

func (te *tableEngine) emitTablePlayerReservedEvent(player *TablePlayerState) {
	// emit event
	// fmt.Printf("->emit player reserved Event: %s\n", player.PlayerID)
	te.onTablePlayerReserved(te.table.Meta.CompetitionID, te.table.ID, player)
}

func (te *tableEngine) emitGamePlayerActionEvent(gameAction TablePlayerGameAction) {
	// emit event
	// fmt.Printf("->emit player game action Event: %s %s %d\n", gameAction.PlayerID, gameAction.Action, gameAction.Chips)
	te.onGamePlayerActionUpdated(gameAction)
}

func (te *tableEngine) emitReadyOpenFirstTableGame(gameCount int, playerStates []*TablePlayerState) {
	// emit event
	// fmt.Printf("->emit ready open first table game: %d players\n", len(playerStates))
	te.onReadyOpenFirstTableGame(gameCount, playerStates)
}

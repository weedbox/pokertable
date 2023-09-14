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
	fmt.Printf("->[Table %s][#%d][%d][%s] emit Event: %s\n", te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName)
	te.onTableUpdated(te.table)
}

func (te *tableEngine) emitErrorEvent(eventName string, playerID string, err error) {
	fmt.Printf("->[Table %s][#%d][%d][%s] emit ERROR Event: %s, Error: %v\n", te.table.ID, te.table.UpdateSerial, te.table.State.GameCount, playerID, eventName, err)
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

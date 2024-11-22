package pokertable

type TableEngineCallbacks struct {
	OnTableUpdated            func(table *Table)
	OnTableErrorUpdated       func(table *Table, err error)
	OnTableStateUpdated       func(event string, table *Table)
	OnTablePlayerStateUpdated func(competitionID, tableID string, playerState *TablePlayerState)
	OnTablePlayerReserved     func(competitionID, tableID string, playerState *TablePlayerState)
	OnGamePlayerActionUpdated func(gameAction TablePlayerGameAction)
	OnAutoGameOpenEnd         func(competitionID, tableID string)
	OnReadyOpenFirstTableGame func(gameCount int, playerStates []*TablePlayerState)
}

func NewTableEngineCallbacks() *TableEngineCallbacks {
	return &TableEngineCallbacks{
		OnTableUpdated:            func(table *Table) {},
		OnTableErrorUpdated:       func(table *Table, err error) {},
		OnTableStateUpdated:       func(event string, table *Table) {},
		OnTablePlayerStateUpdated: func(competitionID, tableID string, playerState *TablePlayerState) {},
		OnTablePlayerReserved:     func(competitionID, tableID string, playerState *TablePlayerState) {},
		OnGamePlayerActionUpdated: func(gameAction TablePlayerGameAction) {},
		OnAutoGameOpenEnd:         func(competitionID, tableID string) {},
		OnReadyOpenFirstTableGame: func(gameCount int, playerStates []*TablePlayerState) {},
	}
}

type TableEngineOptions struct {
	GameContinueInterval int
	OpenGameTimeout      int
}

func NewTableEngineOptions() *TableEngineOptions {
	return &TableEngineOptions{
		GameContinueInterval: 1, // 1 second by default
		OpenGameTimeout:      2,
	}
}

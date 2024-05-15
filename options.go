package pokertable

type TableEngineCallbacks struct {
	OnTableUpdated            func(table *Table)
	OnTableErrorUpdated       func(table *Table, err error)
	OnTableStateUpdated       func(event string, table *Table)
	OnTablePlayerStateUpdated func(competitionID, tableID string, playerState *TablePlayerState)
	OnTablePlayerReserved     func(competitionID, tableID string, playerState *TablePlayerState)
	OnGamePlayerActionUpdated func(gameAction TablePlayerGameAction)
	OnAutoGameOpenEnd         func(competitionID, tableID string)
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
	}
}

type TableEngineOptions struct {
	Interval int
}

func NewTableEngineOptions() *TableEngineOptions {
	return &TableEngineOptions{
		Interval: 0, // 0 second by default
	}
}

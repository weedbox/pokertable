package pokertable

type TableEngineCallbacks struct {
	OnTableUpdated            func(t *Table)
	OnTableErrorUpdated       func(t *Table, err string)
	OnTableStateUpdated       func(string, *Table)
	OnTablePlayerStateUpdated func(string, string, *TablePlayerState)
	OnTablePlayerReserved     func(string, string, *TablePlayerState)
	OnGamePlayerActionUpdated func(TablePlayerGameAction)
}

func NewTableEngineCallbacks() *TableEngineCallbacks {
	return &TableEngineCallbacks{
		OnTableUpdated:            func(*Table) {},
		OnTableErrorUpdated:       func(*Table, string) {},
		OnTableStateUpdated:       func(string, *Table) {},
		OnTablePlayerStateUpdated: func(string, string, *TablePlayerState) {},
		OnTablePlayerReserved:     func(string, string, *TablePlayerState) {},
		OnGamePlayerActionUpdated: func(TablePlayerGameAction) {},
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

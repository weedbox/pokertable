package pokertable

type TableEngineOptions struct {
	Interval                  int
	OnTableUpdated            func(t *Table)
	OnTableErrorUpdated       func(t *Table, err error)
	OnTableStateUpdated       func(string, *Table)
	OnTablePlayerStateUpdated func(string, string, *TablePlayerState)
	OnTablePlayerReserved     func(competitionID string, playerState *TablePlayerState)
}

func NewTableEngineOptions() *TableEngineOptions {
	return &TableEngineOptions{
		Interval:                  0, // 0 second by default
		OnTableUpdated:            func(*Table) {},
		OnTableErrorUpdated:       func(*Table, error) {},
		OnTableStateUpdated:       func(string, *Table) {},
		OnTablePlayerStateUpdated: func(string, string, *TablePlayerState) {},
		OnTablePlayerReserved:     func(string, *TablePlayerState) {},
	}
}

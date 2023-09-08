package pokertable

import (
	"errors"
	"sync"
)

var (
	ErrManagerTableNotFound = errors.New("manager: table not found")
)

type Manager interface {
	// TableEngine Actions
	GetTableEngine(tableID string) (TableEngine, error)
	CreateTable(engineOptions *TableEngineOptions, tableSetting TableSetting) (*Table, error)
	BalanceTable(tableID string) error
	CloseTable(tableID string) error
	StartTableGame(tableID string) error
	TableGameOpen(tableID string) error
	UpdateBlind(tableID string, level int, ante, dealer, sb, bb int64) error

	// Player Table Actions
	PlayerReserve(tableID string, joinPlayer JoinPlayer) error
	PlayersBatchReserve(tableID string, joinPlayers []JoinPlayer) error
	PlayerJoin(tableID, playerID string) error
	PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error
	PlayersLeave(tableID string, playerIDs []string) error

	// Player Game Actions
	PlayerReady(tableID, playerID string) error
	PlayerPay(tableID, playerID string, chips int64) error
	PlayerBet(tableID, playerID string, chips int64) error
	PlayerRaise(tableID, playerID string, chipLevel int64) error
	PlayerCall(tableID, playerID string) error
	PlayerAllin(tableID, playerID string) error
	PlayerCheck(tableID, playerID string) error
	PlayerFold(tableID, playerID string) error
	PlayerPass(tableID, playerID string) error
}

type manager struct {
	// tableEngines map[string]TableEngine
	tableEngines sync.Map
}

func NewManager() Manager {
	return &manager{
		// tableEngines: make(map[string]TableEngine),
		tableEngines: sync.Map{},
	}
}

func (m *manager) GetTableEngine(tableID string) (TableEngine, error) {
	// tableEngine, exist := m.tableEngines[tableID]
	tableEngine, exist := m.tableEngines.Load(tableID)
	if !exist {
		return nil, ErrManagerTableNotFound
	}
	// return tableEngine, nil
	return tableEngine.(TableEngine), nil
}

func (m *manager) CreateTable(engineOptions *TableEngineOptions, tableSetting TableSetting) (*Table, error) {
	var options *TableEngineOptions
	if engineOptions != nil {
		options = engineOptions
	} else {
		options = NewTableEngineOptions()
		options.Interval = 1
	}

	gameBackend := NewNativeGameBackend()
	tableEngine := NewTableEngine(options, WithGameBackend(gameBackend))
	tableEngine.OnTableUpdated(engineOptions.OnTableUpdated)
	tableEngine.OnTableErrorUpdated(engineOptions.OnTableErrorUpdated)
	tableEngine.OnTableStateUpdated(engineOptions.OnTableStateUpdated)
	tableEngine.OnTablePlayerStateUpdated(engineOptions.OnTablePlayerStateUpdated)
	table, err := tableEngine.CreateTable(tableSetting)
	if err != nil {
		return nil, err
	}

	// m.tableEngines[table.ID] = tableEngine
	m.tableEngines.Store(table.ID, tableEngine)
	return table, nil
}

func (m *manager) BalanceTable(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.BalanceTable()
}

func (m *manager) CloseTable(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	if err := tableEngine.CloseTable(); err != nil {
		return err
	}

	// delete(m.tableEngines, tableID)
	m.tableEngines.Delete(tableID)
	return nil
}

func (m *manager) StartTableGame(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.StartTableGame()
}

func (m *manager) TableGameOpen(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.TableGameOpen()
}

func (m *manager) UpdateBlind(tableID string, level int, ante, dealer, sb, bb int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	tableEngine.UpdateBlind(level, ante, dealer, sb, bb)
	return nil
}

func (m *manager) PlayerReserve(tableID string, joinPlayer JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerReserve(joinPlayer)
}

func (m *manager) PlayersBatchReserve(tableID string, joinPlayers []JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayersBatchReserve(joinPlayers)
}

func (m *manager) PlayerJoin(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerJoin(playerID)
}

func (m *manager) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerRedeemChips(joinPlayer)
}

func (m *manager) PlayersLeave(tableID string, playerIDs []string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayersLeave(playerIDs)
}

func (m *manager) PlayerReady(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerReady(playerID)
}

func (m *manager) PlayerPay(tableID, playerID string, chips int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerPay(playerID, chips)
}

func (m *manager) PlayerBet(tableID, playerID string, chips int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerBet(playerID, chips)
}

func (m *manager) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerRaise(playerID, chipLevel)
}

func (m *manager) PlayerCall(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerCall(playerID)
}

func (m *manager) PlayerAllin(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerAllin(playerID)
}

func (m *manager) PlayerCheck(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerCheck(playerID)
}

func (m *manager) PlayerFold(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerFold(playerID)
}

func (m *manager) PlayerPass(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrManagerTableNotFound
	}

	return tableEngine.PlayerPass(playerID)
}

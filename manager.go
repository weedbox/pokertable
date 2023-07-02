package pokertable

import (
	"errors"
	"sync"
)

var (
	ErrTableNotFound = errors.New("table not found")
)

type Manager interface {
	// TableEngine Actions
	GetTableEngine(tableID string) (TableEngine, error)
	CreateTable(tableSetting TableSetting) (*Table, error)
	BalanceTable(tableID string) error
	CloseTable(tableID string) error
	StartTableGame(tableID string) error
	TableGameOpen(tableID string) error

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
	tableEngines sync.Map
}

func NewManager() Manager {
	return &manager{
		tableEngines: sync.Map{},
	}
}

func (m *manager) GetTableEngine(tableID string) (TableEngine, error) {
	tableEngine, exist := m.tableEngines.Load(tableID)
	if !exist {
		return nil, ErrTableNotFound
	}
	return tableEngine.(TableEngine), nil
}

func (m *manager) CreateTable(tableSetting TableSetting) (*Table, error) {
	options := NewTableEngineOptions()
	options.Interval = 1
	gameBackend := NewGameBackend()
	tableEngine := NewTableEngine(options, WithGameBackend(*gameBackend))
	table, err := tableEngine.CreateTable(tableSetting)
	if err != nil {
		return nil, err
	}

	m.tableEngines.Store(table.ID, tableEngine)
	return table, nil
}

func (m *manager) BalanceTable(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.BalanceTable()
}

func (m *manager) CloseTable(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	if err := tableEngine.CloseTable(); err != nil {
		return err
	}

	m.tableEngines.Delete(tableID)
	return nil
}

func (m *manager) StartTableGame(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.StartTableGame()
}

func (m *manager) TableGameOpen(tableID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.TableGameOpen()
}

func (m *manager) PlayerReserve(tableID string, joinPlayer JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerReserve(joinPlayer)
}

func (m *manager) PlayersBatchReserve(tableID string, joinPlayers []JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayersBatchReserve(joinPlayers)
}

func (m *manager) PlayerJoin(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerJoin(playerID)
}

func (m *manager) PlayerRedeemChips(tableID string, joinPlayer JoinPlayer) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerRedeemChips(joinPlayer)
}

func (m *manager) PlayersLeave(tableID string, playerIDs []string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayersLeave(playerIDs)
}

func (m *manager) PlayerReady(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerReady(playerID)
}

func (m *manager) PlayerPay(tableID, playerID string, chips int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerPay(playerID, chips)
}

func (m *manager) PlayerBet(tableID, playerID string, chips int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerBet(playerID, chips)
}

func (m *manager) PlayerRaise(tableID, playerID string, chipLevel int64) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerRaise(playerID, chipLevel)
}

func (m *manager) PlayerCall(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerCall(playerID)
}

func (m *manager) PlayerAllin(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerAllin(playerID)
}

func (m *manager) PlayerCheck(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerCheck(playerID)
}

func (m *manager) PlayerFold(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerFold(playerID)
}

func (m *manager) PlayerPass(tableID, playerID string) error {
	tableEngine, err := m.GetTableEngine(tableID)
	if err != nil {
		return ErrTableNotFound
	}

	return tableEngine.PlayerPass(playerID)
}
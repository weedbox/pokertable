package seat_manager

import (
	"errors"
)

var (
	ErrNotEnoughSeats          = errors.New("seat manager: no enough seats")
	ErrPlayerNotFound          = errors.New("seat manager: player not found")
	ErrPlayerIsAlreadyExist    = errors.New("seat manager: player is already exist")
	ErrUnavailableSeat         = errors.New("seat manager: seat is not available")
	ErrDuplicatePlayers        = errors.New("seat manager: duplicate players detected")
	ErrDuplicateSeats          = errors.New("seat manager: duplicate seats detected")
	ErrSeatAlreadyIsTaken      = errors.New("seat manager: seat is already taken")
	ErrUnableToInitPositions   = errors.New("seat manager: unable to init positions")
	ErrAlreadyInitPositions    = errors.New("seat manager: already init positions")
	ErrUnableToRotatePositions = errors.New("seat manager: unable to rotate positions")

	SupportedRules = []string{Rule_Default, Rule_ShortDeck}
)

type SeatManager interface {
	GetSeatID(playerID string) (int, error)
	RandomAssignSeats(playerIDs []string) error
	AssignSeats(playerSeatIDs map[string]int) error
	RemoveSeats(playerIDs []string) error
	UpdatePlayerHasChips(playerID string, hasChips bool) error
	JoinPlayers(playerIDs []string) error
	InitPositions(isRandom bool) error
	RotatePositions() error
	IsPlayerBetweenDealerBB(playerID string) bool

	Seats() map[int]*SeatPlayer
	CurrentDealerSeatID() int
	CurrentSBSeatID() int
	CurrentBBSeatID() int
	IsInitPositions() bool
	IsPlayerActive(playerID string) (bool, error)
	ListPlayerSeatsFromDealer() []*SeatPlayer
}

type SeatPlayer struct {
	ID                string `json:"id"`
	IsIn              bool   `json:"is_in"`
	IsBetweenDealerBB bool   `json:"is_between_dealer_bb"`
	HasChips          bool   `json:"has_chips"`
}

func (sp *SeatPlayer) Active() bool {
	return sp.IsIn && !sp.IsBetweenDealerBB && sp.HasChips
}

func NewSeatManager(maxSeats int, rule string) SeatManager {
	seatData := make(map[int]*SeatPlayer)
	for i := 0; i < maxSeats; i++ {
		seatData[i] = nil
	}

	return &seatManager{
		MaxSeat:      maxSeats,
		SeatData:     seatData,
		DealerSeatID: UnsetSeatID,
		SBSeatID:     UnsetSeatID,
		BBSeatID:     UnsetSeatID,
		Rule:         rule,
		IsInit:       false,
	}
}

func NewSeatManagerFromState(sm *seatManager) SeatManager {
	return sm
}

package pokertable

type RequestAction string

const (
	RequestAction_BalanceTable      RequestAction = "BalanceTable"
	RequestAction_DeleteTable       RequestAction = "DeleteTable"
	RequestAction_StartTableGame    RequestAction = "StartTableGame"
	RequestAction_TableGameOpen     RequestAction = "TableGameOpen"
	RequestAction_PlayerJoin        RequestAction = "PlayerJoin"
	RequestAction_PlayerRedeemChips RequestAction = "PlayerRedeemChips"
	RequestAction_PlayersLeave      RequestAction = "PlayersLeave"
	RequestAction_PlayerReady       RequestAction = "PlayerReady"
	RequestAction_PlayerPay         RequestAction = "PlayerPay"
	RequestAction_PlayerBet         RequestAction = "PlayerBet"
	RequestAction_PlayerRaise       RequestAction = "PlayerRaise"
	RequestAction_PlayerCall        RequestAction = "PlayerCall"
	RequestAction_PlayerAllin       RequestAction = "PlayerAllin"
	RequestAction_PlayerCheck       RequestAction = "PlayerCheck"
	RequestAction_PlayerFold        RequestAction = "PlayerFold"
	RequestAction_PlayerPass        RequestAction = "PlayerPass"
)

type Request struct {
	Action  RequestAction
	Payload Payload
}

type Payload struct {
	TableGame *TableGame
	Param     interface{}
}

type PlayerPayParam struct {
	PlayerID string
	Chips    int64
}

type PlayerBetParam struct {
	PlayerID string
	Chips    int64
}

type PlayerRaiseParam struct {
	PlayerID  string
	ChipLevel int64
}

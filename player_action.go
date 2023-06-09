package pokertable

import (
	"github.com/thoas/go-funk"
)

/*
	PlayerJoin 玩家入桌
	  - 適用時機:
	    - 報名入桌
		- 補碼入桌
*/
func (te *tableEngine) PlayerJoin(table Table, joinPlayer JoinPlayer) (Table, error) {
	// find player index in PlayerStates
	targetPlayerIdx := te.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)

	// do logic
	if targetPlayerIdx == UnsetValue {
		if len(table.State.PlayerStates) == table.Meta.CompetitionMeta.TableMaxSeatCount {
			return table, ErrNoEmptySeats
		}

		// BuyIn
		player := TablePlayerState{
			PlayerID:          joinPlayer.PlayerID,
			SeatIndex:         UnsetValue,
			Positions:         []string{Position_Unknown},
			IsParticipated:    true,
			IsBetweenDealerBB: false,
			Bankroll:          joinPlayer.RedeemChips,
		}
		table.State.PlayerStates = append(table.State.PlayerStates, &player)

		// update seat
		newPlayerIdx := len(table.State.PlayerStates) - 1
		seatIdx := RandomSeatIndex(table.State.PlayerSeatMap)
		table.State.PlayerSeatMap[seatIdx] = newPlayerIdx
		table.State.PlayerStates[newPlayerIdx].SeatIndex = seatIdx
		table.State.PlayerStates[newPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(seatIdx, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
	} else {
		// ReBuy
		// 補碼要檢查玩家是否介於 Dealer-BB 之間
		table.State.PlayerStates[targetPlayerIdx].IsBetweenDealerBB = IsBetweenDealerBB(table.State.PlayerStates[targetPlayerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
		table.State.PlayerStates[targetPlayerIdx].Bankroll += joinPlayer.RedeemChips
		table.State.PlayerStates[targetPlayerIdx].IsParticipated = true
	}

	return table, nil
}

/*
	PlayerRedeemChips 增購籌碼
	  - 適用時機:
	    - 增購
*/
func (te *tableEngine) PlayerRedeemChips(table Table, joinPlayer JoinPlayer) (Table, error) {
	// find player index in PlayerStates
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, joinPlayer.PlayerID)
	if playerIdx == UnsetValue {
		return table, ErrPlayerNotFound
	}

	// do logic
	// 如果是 Bankroll 為 0 的情況，增購要檢查玩家是否介於 Dealer-BB 之間
	if table.State.PlayerStates[playerIdx].Bankroll == 0 {
		table.State.PlayerStates[playerIdx].IsBetweenDealerBB = IsBetweenDealerBB(table.State.PlayerStates[playerIdx].SeatIndex, table.State.CurrentDealerSeatIndex, table.State.CurrentBBSeatIndex, table.Meta.CompetitionMeta.TableMaxSeatCount, table.Meta.CompetitionMeta.Rule)
	}
	table.State.PlayerStates[playerIdx].Bankroll += joinPlayer.RedeemChips

	return table, nil
}

/*
	PlayerLeave 玩家們離開桌次
	  - 適用時機:
	    - CT 退桌 (玩家有籌碼)
		- CT 放棄補碼 (玩家沒有籌碼)
	    - CT/MTT 斷線且補碼中時(視為淘汰離開)
	    - CASH 離開 (準備結算)
		- CT/MTT 停止買入後被淘汰
*/
func (te *tableEngine) PlayersLeave(table Table, playerIDs []string) Table {
	// find player index in PlayerStates
	leavePlayerIndexes := make([]int, 0)
	for _, playerID := range playerIDs {
		playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
		if playerIdx != UnsetValue {
			leavePlayerIndexes = append(leavePlayerIndexes, playerIdx)
		}
	}

	if len(leavePlayerIndexes) == 0 {
		return table
	}

	// do logic
	// set leave PlayerIdx int seatMap to UnsetValue
	leavePlayerIDMap := make(map[string]interface{})
	for _, leavePlayerIdx := range leavePlayerIndexes {
		leavePlayer := table.State.PlayerStates[leavePlayerIdx]
		leavePlayerIDMap[leavePlayer.PlayerID] = struct{}{}
		table.State.PlayerSeatMap[leavePlayer.SeatIndex] = UnsetValue
	}

	// delete target players in PlayerStates
	table.State.PlayerStates = funk.Filter(table.State.PlayerStates, func(player *TablePlayerState) bool {
		_, exist := leavePlayerIDMap[player.PlayerID]
		return !exist
	}).([]*TablePlayerState)

	// update current PlayerSeatMap player indexes in PlayerSeatMap
	for newPlayerIdx, player := range table.State.PlayerStates {
		table.State.PlayerSeatMap[player.SeatIndex] = newPlayerIdx
	}

	return table
}

func (te *tableEngine) PlayerReady(table Table, playerID string) (Table, error) {
	// find playing player index
	playingPlayerIdx := te.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return table, ErrPlayerNotFound
	}

	// do ready
	gameState, err := te.gameEngine.PlayerReady(playingPlayerIdx)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerReady] [%s] %s ready error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	te.logger.Debugf("[tableEngine#PlayerReady] [%s] %s is ready. CurrentEvent: %s\n", gameState.Status.Round, playerID, gameState.Status.CurrentEvent.Name)
	table.State.GameState = gameState

	return table, nil
}

func (te *tableEngine) PlayerPay(table Table, playerID string, chips int64) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Pay(chips)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPay] [%s] %s pay(%d) error: %+v\n", gameState.Status.Round, playerID, chips, err)
		return table, err
	}
	table.State.GameState = gameState
	te.logger.Debug("[tableEngine#PlayerPay] dealer receive %d.")

	return table, nil

}

func (te *tableEngine) PlayerPayAnte(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.PayAnte()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPay] [%s] %s pay ante error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState
	te.logger.Debug("[tableEngine#PlayerPayAnte] dealer receive ante from all players.")

	return table, nil
}

func (te *tableEngine) PlayerPaySB(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.PaySB()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPaySB] [%s] %s pay sb error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState
	te.logger.Debug("[tableEngine#PlayerPaySB] dealer receive sb.")

	return table, nil
}

func (te *tableEngine) PlayerPayBB(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.PayBB()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerPayBB] [%s] %s pay bb error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState
	te.logger.Debug("[tableEngine#PlayerPaySB] dealer receive bb.")

	return table, nil
}

func (te *tableEngine) PlayerBet(table Table, playerID string, chips int64) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Bet(chips)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerBet] [%s] %s bet(%d) error: %+v\n", gameState.Status.Round, playerID, chips, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerBet] [%s] %s(%+v) bet(%d)\n", gameState.Status.Round, playerID, positions, chips)

	return table, nil
}

func (te *tableEngine) PlayerRaise(table Table, playerID string, chipLevel int64) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Raise(chipLevel)
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerRaise] [%s] %s raise(%d) error: %+v\n", gameState.Status.Round, playerID, chipLevel, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerRaise] [%s] %s(%+v) raise(%d)\n", gameState.Status.Round, playerID, positions, chipLevel)

	return table, nil
}

func (te *tableEngine) PlayerCall(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Call()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerCall] [%s] %s call error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerCall] [%s] %s(%+v) call\n", gameState.Status.Round, playerID, positions)

	return table, nil
}

func (te *tableEngine) PlayerAllin(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Allin()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerAllin] [%s] %s allin error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerAllin] [%s] %s(%+v) allin\n", gameState.Status.Round, playerID, positions)

	return table, nil
}

func (te *tableEngine) PlayerCheck(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Check()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerCheck] [%s] %s check error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerCheck] [%s] %s(%+v) check\n", gameState.Status.Round, playerID, positions)

	return table, nil
}

func (te *tableEngine) PlayerFold(table Table, playerID string) (Table, error) {
	// validate player action
	if err := te.validatePlayerMove(table, playerID); err != nil {
		return table, err
	}

	// do action
	gameState, err := te.gameEngine.Fold()
	if err != nil {
		te.logger.Debugf("[tableEngine#PlayerFold] [%s] %s fold error: %+v\n", gameState.Status.Round, playerID, err)
		return table, err
	}
	table.State.GameState = gameState

	// debug log
	positions := make([]string, 0)
	playerIdx := te.findPlayerIdx(table.State.PlayerStates, playerID)
	if playerIdx != UnsetValue {
		positions = table.State.PlayerStates[playerIdx].Positions
	}
	te.logger.Debugf("[tableEngine#PlayerFold] [%s] %s(%+v) fold\n", gameState.Status.Round, playerID, positions)

	return table, nil
}

func (te *tableEngine) validatePlayerMove(table Table, playerID string) error {
	// find playing player index
	playingPlayerIdx := te.findPlayingPlayerIdx(table.State.PlayerStates, table.State.PlayingPlayerIndexes, playerID)
	if playingPlayerIdx == UnsetValue {
		return ErrPlayerNotFound
	}

	// check if player can do action
	if te.gameEngine.GameState().Status.CurrentPlayer != playingPlayerIdx {
		return ErrPlayerInvalidAction
	}

	return nil
}

func (te *tableEngine) findPlayingPlayerIdx(players []*TablePlayerState, playingPlayerIndexes []int, targetPlayerID string) int {
	for idx, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		if player.PlayerID == targetPlayerID {
			return idx
		}
	}
	return UnsetValue
}

func (te *tableEngine) findPlayerIdx(players []*TablePlayerState, targetPlayerID string) int {
	for idx, player := range players {
		if player.PlayerID == targetPlayerID {
			return idx
		}
	}

	return UnsetValue
}

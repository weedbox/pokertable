package pokertable

import (
	"fmt"
	"strconv"
	"time"
)

func (engine tableEngine) debugPrintTable(message string, table Table) {
	timeString := func(timestamp int64) string {
		return time.Unix(timestamp, 0).Format("2006-01-02 15:04:0")
	}

	boolToString := func(value bool) string {
		if value {
			return "O"
		} else {
			return "X"
		}
	}

	fmt.Printf("---------- [%s] ----------\n", message)
	fmt.Println("[Table ID] ", table.ID)
	fmt.Println("[Table StartAt] ", timeString(table.State.StartGameAt))
	fmt.Println("[Table Game Count] ", table.State.GameCount)
	fmt.Println("[Table Players]")
	for _, player := range table.State.PlayerStates {
		seat := "X"
		if player.SeatIndex != -1 {
			seat = strconv.Itoa(player.SeatIndex)
		}
		fmt.Printf("seat: %s [%v], participated: %s, player: %s\n", seat, player.Positions, boolToString(player.IsParticipated), player.PlayerID)
	}

	if table.State.CurrentDealerSeatIndex != -1 {
		dealerPlayerIndex := table.State.PlayerSeatMap[table.State.CurrentDealerSeatIndex]
		if dealerPlayerIndex == -1 {
			fmt.Println("[Table Current Dealer] X")
		} else {
			fmt.Println("[Table Current Dealer] ", table.State.PlayerStates[dealerPlayerIndex].PlayerID)
		}
	} else {
		fmt.Println("[Table Current Dealer] X")
	}

	if table.State.CurrentBBSeatIndex != -1 {
		bbPlayerIndex := table.State.PlayerSeatMap[table.State.CurrentBBSeatIndex]
		if bbPlayerIndex == -1 {
			fmt.Println("[Table Current BB] X")
		} else {
			fmt.Println("[Table Current BB] ", table.State.PlayerStates[bbPlayerIndex].PlayerID)
		}
	} else {
		fmt.Println("[Table Current BB] X")
	}

	fmt.Printf("[Table SeatMap] %+v\n", table.State.PlayerSeatMap)
	for seatIndex, playerIndex := range table.State.PlayerSeatMap {
		playerID := "X"
		positions := []string{"Unknown"}
		bankroll := "X"
		isBetweenDealerBB := "X"
		if playerIndex != -1 {
			playerID = table.State.PlayerStates[playerIndex].PlayerID
			positions = table.State.PlayerStates[playerIndex].Positions
			bankroll = fmt.Sprintf("%d", table.State.PlayerStates[playerIndex].Bankroll)
			isBetweenDealerBB = fmt.Sprintf("%v", table.State.PlayerStates[playerIndex].IsBetweenDealerBB)
		}

		fmt.Printf("seat: %d, position: %v, player: %s, bankroll: %s, between bb-dealer? %s\n", seatIndex, positions, playerID, bankroll, isBetweenDealerBB)
	}

	fmt.Println("[Blind Data]")
	fmt.Println("InitialLevel: ", table.State.BlindState.InitialLevel)
	fmt.Printf("CurrentLevel: %+v\n", table.State.BlindState.CurrentBlindLevel())
	fmt.Printf("CurrentLevel EndAt: %+v\n", timeString(table.State.BlindState.CurrentBlindLevel().LevelEndAt))
	fmt.Println("CurrentLevelIndex: ", table.State.BlindState.CurrentLevelIndex)
	fmt.Println("FinalBuyInLevelIndex: ", table.State.BlindState.FinalBuyInLevelIndex)
	for _, blindLevelState := range table.State.BlindState.LevelStates {
		blindLevel := blindLevelState
		level := strconv.Itoa(blindLevel.Level)
		if blindLevel.Level == -1 {
			level = "中場休息"
		}

		endAt := "X"
		if blindLevelState.LevelEndAt != -1 {
			endAt = timeString(blindLevelState.LevelEndAt)
		}

		fmt.Printf("Level: %s, (sb,bb,ante): (%d,%d,%d), end: %s\n", level, blindLevel.SBChips, blindLevel.BBChips, blindLevel.AnteChips, endAt)
	}

	fmt.Println("[Playing Players]")
	for _, playerIdx := range table.State.PlayingPlayerIndexes {
		player := table.State.PlayerStates[playerIdx]
		seat := "X"
		if player.SeatIndex != -1 {
			seat = strconv.Itoa(player.SeatIndex)
		}
		fmt.Printf("seat: %s [%v], player: %s, bankroll: %d, between bb-dealer? %v\n", seat, player.Positions, player.PlayerID, player.Bankroll, player.IsBetweenDealerBB)
	}

	fmt.Println()
}

func (engine tableEngine) debugPrintGameStateResult(table Table) {
	playerIDMapper := func(table Table, gameStatePlayerIndex int) string {
		for playingPlayerIndex, playerIndex := range table.State.PlayingPlayerIndexes {
			if playingPlayerIndex == gameStatePlayerIndex {
				return table.State.PlayerStates[playerIndex].PlayerID
			}
		}
		return ""
	}

	fmt.Println("---------- Game Result ----------")
	result := table.State.GameState.Result
	for _, player := range result.Players {
		playerID := playerIDMapper(table, player.Idx)
		fmt.Printf("%s: final: %d, changed: %d\n", playerID, player.Final, player.Changed)
	}
	for idx, pot := range result.Pots {
		fmt.Printf("pot[%d]: %d\n", idx, pot.Total)
		for _, winner := range pot.Winners {
			playerID := playerIDMapper(table, winner.Idx)
			fmt.Printf("%s: withdraw: %d\n", playerID, winner.Withdraw)
		}
		fmt.Println()
	}

	fmt.Println("---------- Player Result ----------")
	for _, player := range table.State.PlayerStates {
		fmt.Printf("%s: seat: %d[%+v], bankroll: %d\n", player.PlayerID, player.SeatIndex, player.Positions, player.Bankroll)
	}
}

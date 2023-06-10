package pokertable

import (
	"fmt"
	"strconv"
	"time"
)

func (t Table) debugPrintTable(message string) {
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
	fmt.Println("[Table ID] ", t.ID)
	fmt.Println("[Table StartAt] ", timeString(t.State.StartGameAt))
	fmt.Println("[Table Game Count] ", t.State.GameCount)
	fmt.Println("[Table Players]")
	for _, player := range t.State.PlayerStates {
		seat := "X"
		if player.SeatIndex != -1 {
			seat = strconv.Itoa(player.SeatIndex)
		}
		fmt.Printf("seat: %s [%v], participated: %s, player: %s\n", seat, player.Positions, boolToString(player.IsParticipated), player.PlayerID)
	}

	if t.State.CurrentDealerSeatIndex != -1 {
		dealerPlayerIndex := t.State.PlayerSeatMap[t.State.CurrentDealerSeatIndex]
		if dealerPlayerIndex == -1 {
			fmt.Println("[Table Current Dealer] X")
		} else {
			fmt.Println("[Table Current Dealer] ", t.State.PlayerStates[dealerPlayerIndex].PlayerID)
		}
	} else {
		fmt.Println("[Table Current Dealer] X")
	}

	if t.State.CurrentBBSeatIndex != -1 {
		bbPlayerIndex := t.State.PlayerSeatMap[t.State.CurrentBBSeatIndex]
		if bbPlayerIndex == -1 {
			fmt.Println("[Table Current BB] X")
		} else {
			fmt.Println("[Table Current BB] ", t.State.PlayerStates[bbPlayerIndex].PlayerID)
		}
	} else {
		fmt.Println("[Table Current BB] X")
	}

	fmt.Printf("[Table SeatMap] %+v\n", t.State.PlayerSeatMap)
	for seatIndex, playerIndex := range t.State.PlayerSeatMap {
		playerID := "X"
		positions := []string{"Unknown"}
		bankroll := "X"
		isBetweenDealerBB := "X"
		if playerIndex != -1 {
			playerID = t.State.PlayerStates[playerIndex].PlayerID
			positions = t.State.PlayerStates[playerIndex].Positions
			bankroll = fmt.Sprintf("%d", t.State.PlayerStates[playerIndex].Bankroll)
			isBetweenDealerBB = fmt.Sprintf("%v", t.State.PlayerStates[playerIndex].IsBetweenDealerBB)
		}

		fmt.Printf("seat: %d, position: %v, player: %s, bankroll: %s, between bb-dealer? %s\n", seatIndex, positions, playerID, bankroll, isBetweenDealerBB)
	}

	fmt.Println("[Blind Data]")
	fmt.Println("InitialLevel: ", t.State.BlindState.InitialLevel)
	fmt.Printf("CurrentLevel: %+v\n", t.State.BlindState.CurrentBlindLevel())
	fmt.Printf("CurrentLevel EndAt: %+v\n", timeString(t.State.BlindState.CurrentBlindLevel().LevelEndAt))
	fmt.Println("CurrentLevelIndex: ", t.State.BlindState.CurrentLevelIndex)
	fmt.Println("FinalBuyInLevelIndex: ", t.State.BlindState.FinalBuyInLevelIndex)
	for _, blindLevelState := range t.State.BlindState.LevelStates {
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

	fmt.Println("[Game Players]")
	for _, playerIdx := range t.State.GamePlayerIndexes {
		player := t.State.PlayerStates[playerIdx]
		seat := "X"
		if player.SeatIndex != -1 {
			seat = strconv.Itoa(player.SeatIndex)
		}
		fmt.Printf("seat: %s [%v], player: %s, bankroll: %d, between bb-dealer? %v\n", seat, player.Positions, player.PlayerID, player.Bankroll, player.IsBetweenDealerBB)
	}

	fmt.Println()
}

func (t Table) debugPrintGameStateResult() {
	playerIDMapper := func(t Table, gameStatePlayerIdx int) string {
		for gamePlayerIdx, playerIdx := range t.State.GamePlayerIndexes {
			if gamePlayerIdx == gameStatePlayerIdx {
				return t.State.PlayerStates[playerIdx].PlayerID
			}
		}
		return ""
	}

	fmt.Println("---------- Game Result ----------")
	result := t.State.GameState.Result
	for _, player := range result.Players {
		playerID := playerIDMapper(t, player.Idx)
		fmt.Printf("%s: final: %d, changed: %d\n", playerID, player.Final, player.Changed)
	}
	for idx, pot := range result.Pots {
		fmt.Printf("pot[%d]: %d\n", idx, pot.Total)
		for _, winner := range pot.Winners {
			playerID := playerIDMapper(t, winner.Idx)
			fmt.Printf("%s: withdraw: %d\n", playerID, winner.Withdraw)
		}
		fmt.Println()
	}

	fmt.Println("---------- Player Result ----------")
	for _, player := range t.State.PlayerStates {
		fmt.Printf("%s: seat: %d[%+v], bankroll: %d\n", player.PlayerID, player.SeatIndex, player.Positions, player.Bankroll)
	}
}

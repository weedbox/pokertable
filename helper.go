package pokertable

import (
	"time"

	"github.com/weedbox/pokerface"
)

func (engine *tableEngine) getGameStateEventName(event pokerface.GameEvent) string {
	return pokerface.GameEventSymbols[event]
}

func (engine *tableEngine) newGameEngineSetting(rule string, blind TableBlindLevelState, dealerBlindTimes int, players []*TablePlayerState, playingPlayerIndexes []int) GameEngineSetting {
	setting := GameEngineSetting{
		Rule: rule,
		Ante: blind.AnteChips,
		Blind: pokerface.BlindSetting{
			Dealer: blind.AnteChips * (int64(dealerBlindTimes) - 1),
			SB:     blind.SBChips,
			BB:     blind.BBChips,
		},
	}
	playerSettings := make([]*pokerface.PlayerSetting, 0)
	for _, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		playerSettings = append(playerSettings, &pokerface.PlayerSetting{
			Bankroll:  player.Bankroll,
			Positions: player.Positions,
		})
	}
	setting.Players = playerSettings
	return setting
}

func (engine *tableEngine) findPlayerIdx(players []*TablePlayerState, targetPlayerID string) int {
	for idx, player := range players {
		if player.PlayerID == targetPlayerID {
			return idx
		}
	}

	return UnsetValue
}

func (engine *tableEngine) findPlayingPlayerIdx(players []*TablePlayerState, playingPlayerIndexes []int, targetPlayerID string) int {
	for idx, playerIdx := range playingPlayerIndexes {
		player := players[playerIdx]
		if player.PlayerID == targetPlayerID {
			return idx
		}
	}
	return UnsetValue
}

/*
	isTableClose 計算本桌是否已結束
	  - 結束條件 1: 達到賽局結束時間
	  - 結束條件 2: 停止買入後且存活玩家剩餘 1 人
*/
func (engine *tableEngine) isTableClose(endGameAt int64, alivePlayers []*TablePlayerState, isFinalBuyInLevel bool) bool {
	return time.Now().Unix() > endGameAt || (isFinalBuyInLevel && len(alivePlayers) == 1)
}

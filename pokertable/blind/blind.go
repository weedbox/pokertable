package blind

import (
	"time"

	"github.com/weedbox/pokermodel"
)

type Blind struct{}

func NewBlind() Blind {
	return Blind{}
}

func (blind Blind) ActivateBlindState(startGameAt int64, blindState pokermodel.TableBlindState) pokermodel.TableBlindState {
	for idx, levelState := range blindState.LevelStates {
		if levelState.BlindLevel.Level == blindState.InitialLevel {
			blindState.CurrentLevelIndex = idx
			break
		}
	}
	blindStartAt := startGameAt
	for i := (blindState.InitialLevel - 1); i < len(blindState.LevelStates); i++ {
		if i == blindState.InitialLevel-1 {
			blindState.LevelStates[i].LevelEndAt = blindStartAt
		} else {
			blindState.LevelStates[i].LevelEndAt = blindState.LevelStates[i-1].LevelEndAt
		}
		blindPassedSeconds := int64((time.Duration(blindState.LevelStates[i].BlindLevel.DurationMins) * time.Minute).Seconds())
		blindState.LevelStates[i].LevelEndAt += blindPassedSeconds
	}
	return blindState
}

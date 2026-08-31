package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	// Text fields should feel like normal desktop inputs when Backspace/Delete
	// is held. Start repeating after ~0.23s and then erase at ~30 chars/s.
	textRepeatDelayFrames    = 14
	textRepeatIntervalFrames = 2
)

func textRepeatDurationTriggers(duration int) bool {
	if duration <= 0 {
		return false
	}
	if duration == 1 {
		return true
	}
	if duration < textRepeatDelayFrames {
		return false
	}
	return (duration-textRepeatDelayFrames)%textRepeatIntervalFrames == 0
}

func textEraseRepeated() bool {
	return textRepeatDurationTriggers(inpututil.KeyPressDuration(ebiten.KeyBackspace)) ||
		textRepeatDurationTriggers(inpututil.KeyPressDuration(ebiten.KeyDelete))
}

package notify

import (
	"os/exec"
	"runtime"
)

type SoundType int

const (
	SoundPermission SoundType = iota
	SoundQuestion
	SoundError
)

var macSounds = map[SoundType]string{
	SoundPermission: "/System/Library/Sounds/Sosumi.aiff",
	SoundQuestion:   "/System/Library/Sounds/Ping.aiff",
	SoundError:      "/System/Library/Sounds/Basso.aiff",
}

func PlaySound(st SoundType) {
	if runtime.GOOS == "darwin" {
		if path, ok := macSounds[st]; ok {
			exec.Command("afplay", path).Start()
		}
	}
}

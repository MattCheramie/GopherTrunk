package main

import (
	"fmt"

	"github.com/MattCheramie/GopherTrunk/internal/voice/player"
)

// runAudio dispatches the `gophertrunk audio …` subcommands. v1
// exposes only `list`, which prints the audio outputs the player
// backend can route to. oto/v3 does not expose a device picker —
// it always routes to the OS default sink — so the listing is
// short by design. Kept as a subcommand so future backends can
// add real enumeration without breaking the CLI shape.
func runAudio(args []string) {
	rep := newReporter("audio")
	if len(args) == 0 {
		rep.Fatalf(2, "usage: gophertrunk audio list")
	}
	switch args[0] {
	case "list":
		listAudio()
	default:
		rep.Fatalf(2, "unknown audio subcommand: %s", args[0])
	}
}

func listAudio() {
	devs := player.ListDevices()
	if len(devs) == 0 {
		fmt.Println("no audio output devices available")
		return
	}
	fmt.Printf("%-4s  %s\n", "IDX", "DEVICE")
	for i, d := range devs {
		fmt.Printf("%-4d  %s\n", i, d)
	}
	// The list reads like a picker, and an operator with silent audio
	// tried each entry in turn (#1351). Say what it is for.
	fmt.Println()
	fmt.Println("These are the host outputs audio.device can name when audio.enabled is true")
	fmt.Println("(live playback through this computer's speakers). The web console's player")
	fmt.Println("streams through the browser instead and uses none of them. Both, and the")
	fmt.Println("recordings, only carry sound once a call is decoded: with no control-channel")
	fmt.Println("lock or no voice SDR there is nothing to play. GET /api/v1/audio reports")
	fmt.Println("stream_samples_total = 0 in that case.")
}

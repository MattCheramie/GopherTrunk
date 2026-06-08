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
}

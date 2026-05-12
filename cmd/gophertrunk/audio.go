package main

import (
	"fmt"
	"os"

	"github.com/MattCheramie/GopherTrunk/internal/voice/player"
)

// runAudio dispatches the `gophertrunk audio …` subcommands. v1
// exposes only `list`, which prints the audio outputs the player
// backend can route to. oto/v3 does not expose a device picker —
// it always routes to the OS default sink — so the listing is
// short by design. Kept as a subcommand so future backends can
// add real enumeration without breaking the CLI shape.
func runAudio(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: gophertrunk audio list")
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		listAudio()
	default:
		fmt.Fprintf(os.Stderr, "unknown audio subcommand: %s\n", args[0])
		os.Exit(2)
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

package tui

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
	"github.com/MattCheramie/GopherTrunk/internal/tui/panels"
)

// Polling intervals chosen to balance responsiveness against load
// on the daemon. See plan for the rationale.
const (
	pollHealthEvery     = 2 * time.Second
	pollSystemsEvery    = 10 * time.Second
	pollTalkgroupsEvery = 30 * time.Second
	pollActiveEvery     = 1 * time.Second
	pollMetricsEvery    = 5 * time.Second
	pollDevicesEvery    = 10 * time.Second
	pollScannerEvery    = 2 * time.Second
	pollAudioEvery      = 3 * time.Second
	pollRuntimeEvery    = 30 * time.Second
)

func cmdPollHealth(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		h, err := cli.Health(context.Background())
		return pollHealthMsg{h: h, err: err}
	}
}

func cmdPollVersion(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		v, err := cli.Version(context.Background())
		return pollVersionMsg{v: v, err: err}
	}
}

func cmdPollSystems(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		s, err := cli.Systems(context.Background())
		return pollSystemsMsg{s: s, err: err}
	}
}

func cmdPollTalkgroups(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		tg, err := cli.Talkgroups(context.Background())
		return pollTalkgroupsMsg{tg: tg, err: err}
	}
}

func cmdPollActive(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		c, err := cli.ActiveCalls(context.Background())
		return pollActiveMsg{calls: c, err: err}
	}
}

func cmdPollMetrics(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		m, err := cli.Metrics(context.Background())
		return pollMetricsMsg{m: m, err: err}
	}
}

func cmdPollDevices(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		d, err := cli.Devices(context.Background())
		return pollDevicesMsg{devs: d, err: err}
	}
}

func cmdPollScanner(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		s, err := cli.Scanner(context.Background())
		return pollScannerMsg{s: s, err: err}
	}
}

// Scanner-cockpit write Cmds — one per WriteKind.

func cmdScannerSetMode(cli *client.Client, mode, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerSetMode(context.Background(), mode)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerHuntHold(cli *client.Client, system, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerHuntHold(context.Background(), system)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerHuntResume(cli *client.Client, system, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerHuntResume(context.Background(), system)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerHuntRetune(cli *client.Client, system, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerHuntRetune(context.Background(), system)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerConvHold(cli *client.Client, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerConvHold(context.Background())
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerConvResume(cli *client.Client, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerConvResume(context.Background())
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerConvDwell(cli *client.Client, idx int, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerConvDwell(context.Background(), idx)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerConvLockout(cli *client.Client, idx int, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerConvLockout(context.Background(), idx)
		return writeResultMsg{Label: label, Err: err}
	}
}
func cmdScannerConvUnlockout(cli *client.Client, idx int, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ScannerConvUnlockout(context.Background(), idx)
		return writeResultMsg{Label: label, Err: err}
	}
}

// cmdPollRuntime fetches the daemon's read-only runtime snapshot
// — config knobs, output paths, audio device list, etc. Polled
// every 30s since none of these mutate at runtime; rebuilding the
// Settings panel cells more often would just burn CPU.
func cmdPollRuntime(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		r, err := cli.Runtime(context.Background())
		return pollRuntimeMsg{r: r, err: err}
	}
}

// cmdPollAudio fetches the audio cockpit's current state. Used to
// keep the dashboard volume / mute / record indicator in sync with
// any out-of-band mutations (another TUI instance, a curl command,
// etc.).
func cmdPollAudio(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		a, err := cli.AudioStatus(context.Background())
		return pollAudioMsg{a: a, err: err}
	}
}

// cmdSetAudio dispatches a PATCH /api/v1/audio with whichever knobs
// are non-nil. Returns a writeResultMsg with the supplied label so
// the toast surface stays consistent with the other writes.
func cmdSetAudio(cli *client.Client, volume *float32, muted *bool, recording *bool, label string) tea.Cmd {
	return func() tea.Msg {
		_, err := cli.SetAudio(context.Background(), volume, muted, recording)
		return writeResultMsg{Label: label, Err: err}
	}
}

// cmdScannerManualTune adds a runtime VFO channel and forces dwell.
func cmdScannerManualTune(cli *client.Client, freqHz uint32, label, mode, toastLabel string) tea.Cmd {
	return func() tea.Msg {
		_, err := cli.ScannerManualTune(context.Background(), freqHz, label, mode)
		return writeResultMsg{Label: toastLabel, Err: err}
	}
}

func cmdPollHistory(cli *client.Client, f client.HistoryFilter) tea.Cmd {
	return func() tea.Msg {
		rows, err := cli.History(context.Background(), f)
		return pollHistoryMsg{rows: rows, err: err}
	}
}

// scheduleAfter wraps tea.Tick into a one-shot timer that returns
// the supplied tea.Cmd's message. This is the canonical pattern for
// turning periodic polling into a self-rescheduling Cmd.
func scheduleAfter(d time.Duration, c tea.Cmd) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return c()
	})
}

// listenSSE blocks on the next SSE event and returns it as eventMsg.
// On channel close it returns sseDownMsg, prompting the root model
// to reconnect.
func listenSSE(ch <-chan client.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return sseDownMsg{}
		}
		return eventMsg{ev: ev}
	}
}

// connectSSE opens a new SSE stream against cli and returns the
// channel + cancel func via sseUpMsg. The cancel func tears down
// the long-lived request when the model swaps in a new stream.
func connectSSE(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		ch, _ := cli.Stream(ctx)
		return sseUpMsg{ch: ch, cancel: cancel}
	}
}

func cmdMutationStatus(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		s, err := cli.MutationStatus(context.Background())
		return pollMutationStatusMsg{s: s, err: err}
	}
}

// Write-side Cmd builders. Each returns a tea.Cmd that runs the
// HTTP request and surfaces the outcome as writeResultMsg, which
// the root model turns into a toast.

func cmdEndCall(cli *client.Client, deviceSerial, reason, label string) tea.Cmd {
	return func() tea.Msg {
		err := cli.EndCall(context.Background(), deviceSerial, reason)
		return writeResultMsg{Label: label, Err: err}
	}
}

func cmdUpdateTalkgroup(cli *client.Client, id uint32, priority *int, lockout *bool, scan *bool, label string) tea.Cmd {
	return func() tea.Msg {
		_, err := cli.UpdateTalkgroup(context.Background(), id, priority, lockout, scan)
		return writeResultMsg{Label: label, Err: err}
	}
}

func cmdSweepRetention(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		err := cli.SweepRetention(context.Background())
		return writeResultMsg{Label: "retention sweep", Err: err}
	}
}

func cmdResetTone(cli *client.Client, deviceSerial string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ResetToneDevice(context.Background(), deviceSerial)
		return writeResultMsg{Label: "reset tone detector for " + deviceSerial, Err: err}
	}
}

func cmdFetchSystemDetail(cli *client.Client, name string) tea.Cmd {
	return func() tea.Msg {
		s, err := cli.System(context.Background(), name)
		return systemDetailResultMsg{s: s, err: err}
	}
}

func cmdFetchTalkgroupDetail(cli *client.Client, id uint32) tea.Cmd {
	return func() tea.Msg {
		tg, err := cli.Talkgroup(context.Background(), id)
		return talkgroupDetailResultMsg{tg: tg, err: err}
	}
}

// cmdImportUpload reads every queued path from disk and posts a
// multipart upload to the daemon.
func cmdImportUpload(cli *client.Client, paths []string) tea.Cmd {
	return func() tea.Msg {
		files := make([]client.ImportUploadFile, 0, len(paths))
		for _, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				return panels.ImportPreviewArrivedMsg{Err: err}
			}
			files = append(files, client.ImportUploadFile{
				Filename: filepath.Base(p),
				Data:     data,
			})
		}
		preview, err := cli.ImportUpload(context.Background(), files)
		return panels.ImportPreviewArrivedMsg{Preview: preview, Err: err}
	}
}

// cmdImportCommit finalises a staged preview by ID.
func cmdImportCommit(cli *client.Client, id string, force bool) tea.Cmd {
	return func() tea.Msg {
		res, err := cli.ImportCommit(context.Background(), id, force)
		return panels.ImportResultArrivedMsg{Result: res, Err: err}
	}
}

// cmdImportDiscard drops a staged preview without committing.
func cmdImportDiscard(cli *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		err := cli.ImportDiscard(context.Background(), id)
		// Discard result rides the writeResultMsg path so the
		// root model surfaces a toast — same UX as other mutations.
		return writeResultMsg{Label: "import discard", Err: err}
	}
}

// settingsWriteMsg is the unified result of a PATCH /api/v1/settings
// dispatch. The root Update fans it out into a toast (via the
// existing writeResultMsg flow) and an inline error on the Settings
// panel.
type settingsWriteMsg struct {
	Label string
	Field string
	Err   error
}

// cmdUpdateSettings converts a Settings panel edit (dotted field +
// stringified value) into a SettingsPatch and posts it. Wraps both
// outcomes (network success / network failure / client-side
// validation failure) into settingsWriteMsg so the root model can
// surface a toast and re-focus the row in one place.
func cmdUpdateSettings(cli *client.Client, field, value, label string) tea.Cmd {
	return func() tea.Msg {
		patch, perr := buildSettingsPatch(field, value)
		if perr != nil {
			return settingsWriteMsg{Label: label, Field: field, Err: perr}
		}
		_, err := cli.UpdateSettings(context.Background(), patch)
		return settingsWriteMsg{Label: label, Field: field, Err: err}
	}
}

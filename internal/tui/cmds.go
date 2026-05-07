package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/MattCheramie/GopherTrunk/internal/tui/client"
)

// Polling intervals chosen to balance responsiveness against load
// on the daemon. See plan for the rationale.
const (
	pollHealthEvery     = 2 * time.Second
	pollSystemsEvery    = 10 * time.Second
	pollTalkgroupsEvery = 30 * time.Second
	pollActiveEvery     = 1 * time.Second
	pollMetricsEvery    = 5 * time.Second
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

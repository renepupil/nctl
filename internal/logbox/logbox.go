package logbox

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/grafana/loki/pkg/logcli/output"
	"github.com/grafana/loki/pkg/loghttp"
)

var appStyle = lipgloss.NewStyle().Margin(0, 2, 1, 1)

type Msg struct {
	Line string
	Done bool
}

func (r Msg) String() string {
	return r.Line
}

// LogBox renders a scrolling box in a terminal for outputting logs without
// filling up the whole scrollback. On the first line it shows the waitMessage
// with a spinner in front.
type LogBox struct {
	height      int
	waitMessage string
	results     []Msg
	quitting    bool
	spinner     spinner.Model
	interrupt   chan bool
}

// New initializes a LogBox. The interrupt channel will be written to on
// ctrl+c/ctrl+d since it intercepts these.
func New(height int, waitMessage string, interrupt chan bool) LogBox {
	s := spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{})))
	return LogBox{
		height:      height,
		waitMessage: waitMessage,
		spinner:     s,
		results:     []Msg{},
		interrupt:   interrupt,
	}
}

func (lb LogBox) Init() tea.Cmd {
	return lb.spinner.Tick
}

func (lb LogBox) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			lb.interrupt <- true
			return lb, tea.Quit
		default:
			return lb, nil
		}
	case Msg:
		if msg.Done {
			lb.quitting = true
			return lb, tea.Quit
		}
		if len(lb.results) == lb.height {
			lb.results = append(lb.results[1:], msg)
		} else {
			lb.results = append(lb.results, msg)
		}
		return lb, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		lb.spinner, cmd = lb.spinner.Update(msg)
		return lb, cmd
	default:
		return lb, nil
	}
}

func (lb LogBox) View() string {
	var s string
	if !lb.quitting {
		s += lb.spinner.View() + lb.waitMessage
	}

	if lb.quitting {
		s += "✓" + lb.waitMessage
		return appStyle.Render(s)
	}

	if len(lb.results) > 0 {
		s += "\n\n"
	}

	for _, res := range lb.results {
		s += res.String() + "\n"
	}

	return appStyle.Render(s)
}

// Output implements output.LogOutput to send log lines to the tea.Program.
type Output struct {
	*tea.Program
}

func (f *Output) FormatAndPrintln(ts time.Time, lbls loghttp.LabelSet, maxLabelsLen int, line string) {
	timestamp := ts.In(time.Local).Format(time.RFC3339)
	line = strings.TrimSpace(line)

	// we delay the send to the terminal slightly to make the log output look
	// a little smoother. Since our log forwarder only sends at a certain
	// interval (1s), we often receive 10+ log lines at once.
	f.delaySend(time.Millisecond*10, Msg{Line: fmt.Sprintf("%s %s", color.BlueString(timestamp), line)})
}

func (f *Output) WithWriter(w io.Writer) output.LogOutput {
	return f
}

// delaySend delays the sending of the message by the specified duration.
func (f *Output) delaySend(d time.Duration, msg Msg) {
	time.Sleep(d)
	f.Send(msg)
}

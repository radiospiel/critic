// Demo program showing all available short (multi-character) animation types.
// Run with: go run ./internal/tui/animation/examples/short
package main

import (
	"fmt"
	"os"

	"github.com/radiospiel/critic/src/tui/animation"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	anim        *animation.Animation
	animType    animation.ShortType
	speedFactor float64
	quitting    bool
}

func initialModel() model {
	return model{
		anim:        animation.NewShortAnimation(animation.Wave, true, 1.0),
		animType:    animation.Wave,
		speedFactor: 1.0,
	}
}

func (m model) Init() tea.Cmd {
	return m.anim.TickCmd()
}

func (m *model) switchAnim(t animation.ShortType) {
	m.animType = t
	m.anim = animation.NewShortAnimation(t, m.anim.Colored, m.speedFactor)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "c", "C":
			m.anim.Colored = !m.anim.Colored
			return m, nil
		case "+", "=":
			if m.speedFactor > 0.2 {
				m.speedFactor -= 0.1
				m.switchAnim(m.animType)
				return m, m.anim.TickCmd()
			}
			return m, nil
		case "-", "_":
			if m.speedFactor < 3.0 {
				m.speedFactor += 0.1
				m.switchAnim(m.animType)
				return m, m.anim.TickCmd()
			}
			return m, nil
		case "1":
			m.switchAnim(animation.Wave)
			return m, m.anim.TickCmd()
		case "2":
			m.switchAnim(animation.ProgressBar)
			return m, m.anim.TickCmd()
		case "3":
			m.switchAnim(animation.Snake)
			return m, m.anim.TickCmd()
		case "4":
			m.switchAnim(animation.Pulse)
			return m, m.anim.TickCmd()
		case "5":
			m.switchAnim(animation.Scan)
			return m, m.anim.TickCmd()
		case "6":
			m.switchAnim(animation.Bounce)
			return m, m.anim.TickCmd()
		case "7":
			m.switchAnim(animation.Fire)
			return m, m.anim.TickCmd()
		case "8":
			m.switchAnim(animation.Matrix)
			return m, m.anim.TickCmd()
		case "9":
			m.switchAnim(animation.Equalizer)
			return m, m.anim.TickCmd()
		case "0":
			m.switchAnim(animation.Loading)
			return m, m.anim.TickCmd()
		case "r":
			m.switchAnim(animation.Ripple)
			return m, m.anim.TickCmd()
		case "k":
			m.switchAnim(animation.Knight)
			return m, m.anim.TickCmd()
		}
	case animation.TickMsg:
		// Ignore ticks from previous animation
		if msg.Anim != m.anim {
			return m, nil
		}
		m.anim.Tick()
		return m, m.anim.TickCmd()
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Bye!\n"
	}

	colorStatus := "mono"
	if m.anim.Colored {
		colorStatus = "color"
	}

	return fmt.Sprintf("\n  %s\n\n"+
		"  1:Wave 2:Progress 3:Snake 4:Pulse 5:Scan 6:Bounce\n"+
		"  7:Fire 8:Matrix 9:Equalizer 0:Loading r:Ripple k:Knight\n"+
		"  +/-:speed c:color q:quit\n\n"+
		"  %s (%.0f%%, %s)\n",
		m.anim.Render(),
		m.animType.Name(),
		100/m.speedFactor,
		colorStatus)
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

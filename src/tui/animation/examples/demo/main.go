// Demo program showing all available single-cell animation types.
// Run with: go run ./internal/tui/animation/examples/demo
package main

import (
	"fmt"
	"os"

	"github.org/radiospiel/critic/src/tui/animation"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	anim        *animation.Animation
	animType    animation.Type
	speedFactor float64
	quitting    bool
}

func initialModel() model {
	return model{
		anim:        animation.NewSingleCellAnimation(animation.StarBurst, true, 1.0),
		animType:    animation.StarBurst,
		speedFactor: 1.0,
	}
}

func (m model) Init() tea.Cmd {
	return m.anim.TickCmd()
}

func (m *model) switchAnim(t animation.Type) {
	m.animType = t
	m.anim = animation.NewSingleCellAnimation(t, m.anim.Colored, m.speedFactor)
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
			m.switchAnim(animation.BrailleSpinner)
			return m, m.anim.TickCmd()
		case "2":
			m.switchAnim(animation.CircleQuarters)
			return m, m.anim.TickCmd()
		case "3":
			m.switchAnim(animation.DotPulse)
			return m, m.anim.TickCmd()
		case "4":
			m.switchAnim(animation.VerticalBounce)
			return m, m.anim.TickCmd()
		case "5":
			m.switchAnim(animation.StarBurst)
			return m, m.anim.TickCmd()
		case "6":
			m.switchAnim(animation.ExpandingRings)
			return m, m.anim.TickCmd()
		case "7":
			m.switchAnim(animation.BrailleSnake)
			return m, m.anim.TickCmd()
		case "8":
			m.switchAnim(animation.BlockQuadrants)
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
		"  1:Braille 2:Circle 3:Dot 4:Bounce 5:Star 6:Rings 7:Snake 8:Block\n"+
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

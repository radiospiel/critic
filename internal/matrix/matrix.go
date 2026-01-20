package matrix

import (
	"math/rand"

	"git.15b.it/eno/critic/teapot"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Screensaver displays a Matrix-style rain effect with kanji characters.
type Screensaver struct {
	teapot.BaseView

	width    int
	height   int
	streaks  []*rainStreak
	active   bool
	onDone   func() // Callback when screensaver is dismissed
	rng      *rand.Rand
	lastTick int64
}

// rainStreak represents a single streak of falling characters.
// Each streak occupies 2 terminal columns (for double-width characters).
type rainStreak struct {
	xPos      int     // X position (column) where this streak renders
	chars     []rune  // Characters in the streak (from top to bottom)
	head      int     // Position of the "head" (leading character)
	length    int     // Length of the visible trail
	speed     int     // How many ticks between updates
	tickCount int     // Counter for speed
	active    bool    // Whether this streak is currently running
	startDelay int    // Ticks to wait before starting
}

// All characters are full-width (2 columns each) for consistent rendering.
// Using only full-width katakana to avoid width inconsistencies.
var matrixChars = []rune{
	// Full-width Katakana (all 2 columns wide)
	'ア', 'イ', 'ウ', 'エ', 'オ',
	'カ', 'キ', 'ク', 'ケ', 'コ',
	'サ', 'シ', 'ス', 'セ', 'ソ',
	'タ', 'チ', 'ツ', 'テ', 'ト',
	'ナ', 'ニ', 'ヌ', 'ネ', 'ノ',
	'ハ', 'ヒ', 'フ', 'ヘ', 'ホ',
	'マ', 'ミ', 'ム', 'メ', 'モ',
	'ヤ', 'ユ', 'ヨ',
	'ラ', 'リ', 'ル', 'レ', 'ロ',
	'ワ', 'ヲ', 'ン',
	// Full-width numbers (2 columns wide)
	'０', '１', '２', '３', '４', '５', '６', '７', '８', '９',
	// Some full-width symbols
	'＊', '＋', '－', '＝', '＠', '＃',
}

// NewScreensaver creates a new Matrix screensaver
func NewScreensaver() *Screensaver {
	m := &Screensaver{
		BaseView: teapot.NewBaseView(),
		rng:      rand.New(rand.NewSource(rand.Int63())),
	}
	m.SetFocusable(true)
	return m
}

// SetOnDone sets the callback to call when the screensaver is dismissed
func (m *Screensaver) SetOnDone(onDone func()) {
	m.onDone = onDone
}

// Start activates the screensaver
func (m *Screensaver) Start(width, height int) {
	m.width = width
	m.height = height
	m.active = true
	m.lastTick = teapot.GlobalTickCount

	// Calculate number of slots available.
	// Each slot is 2 columns wide (for double-width characters).
	// Slots are at even positions: 0, 2, 4, 6, ...
	numSlots := width / 2
	if numSlots < 1 {
		numSlots = 1
	}

	// Use about half of available slots for good visual density
	numStreaks := numSlots / 2
	if numStreaks < 1 {
		numStreaks = 1
	}

	m.streaks = make([]*rainStreak, numStreaks)

	// Randomly select which slots to use
	slots := make([]int, numSlots)
	for i := range slots {
		slots[i] = i * 2 // even positions: 0, 2, 4, 6, ...
	}
	m.rng.Shuffle(len(slots), func(i, j int) {
		slots[i], slots[j] = slots[j], slots[i]
	})

	for i := 0; i < numStreaks; i++ {
		m.streaks[i] = m.newStreak(slots[i])
		m.streaks[i].startDelay = m.rng.Intn(height)
	}

	teapot.SubscribeToGlobalTicks(m)
}

// Stop deactivates the screensaver
func (m *Screensaver) Stop() {
	m.active = false
	teapot.UnsubscribeFromGlobalTicks(m)
}

// IsActive returns whether the screensaver is currently running
func (m *Screensaver) IsActive() bool {
	return m.active
}

// newStreak creates a new rain streak with random properties at the given x position
func (m *Screensaver) newStreak(xPos int) *rainStreak {
	length := m.height/3 + m.rng.Intn(m.height/2+1)
	if length < 3 {
		length = 3
	}

	chars := make([]rune, m.height+length)
	for i := range chars {
		chars[i] = matrixChars[m.rng.Intn(len(matrixChars))]
	}

	return &rainStreak{
		xPos:      xPos,
		chars:     chars,
		head:      -length, // Start above the screen
		length:    length,
		speed:     1 + m.rng.Intn(2), // 1-2 ticks per move
		tickCount: 0,
		active:    true,
	}
}

// resetStreak resets a streak that has gone off-screen
func (m *Screensaver) resetStreak(streak *rainStreak) {
	streak.head = -streak.length
	streak.speed = 1 + m.rng.Intn(2)
	streak.startDelay = m.rng.Intn(m.height / 2)

	// Pick a new random even position
	numSlots := m.width / 2
	if numSlots < 1 {
		numSlots = 1
	}
	streak.xPos = m.rng.Intn(numSlots) * 2

	// Regenerate characters
	for i := range streak.chars {
		streak.chars[i] = matrixChars[m.rng.Intn(len(matrixChars))]
	}
}

// HandleTick implements teapot.TickHandler
func (m *Screensaver) HandleTick() {
	if !m.active {
		return
	}

	// Update streaks
	for _, streak := range m.streaks {
		if streak.startDelay > 0 {
			streak.startDelay--
			continue
		}

		streak.tickCount++
		if streak.tickCount >= streak.speed {
			streak.tickCount = 0
			streak.head++

			// Occasionally change a random character in the trail
			if m.rng.Intn(10) == 0 {
				idx := m.rng.Intn(len(streak.chars))
				streak.chars[idx] = matrixChars[m.rng.Intn(len(matrixChars))]
			}

			// Reset streak if it's completely off-screen
			if streak.head-streak.length > m.height {
				m.resetStreak(streak)
			}
		}
	}

	m.Repaint()
}

// MightBeDirty always returns true since we're animating
func (m *Screensaver) MightBeDirty() bool {
	return m.active
}

// HandleKey handles keyboard input - any key dismisses the screensaver
func (m *Screensaver) HandleKey(msg tea.KeyMsg) (handled bool, cmd tea.Cmd) {
	if m.active {
		m.Stop()
		if m.onDone != nil {
			m.onDone()
		}
		return true, nil
	}
	return false, nil
}

// Render renders the Matrix rain effect
func (m *Screensaver) Render(buf *teapot.SubBuffer) {
	if !m.active {
		return
	}

	// Fill with black background
	blackCell := teapot.Cell{
		Rune:  ' ',
		Style: lipgloss.NewStyle().Background(lipgloss.Color("0")),
	}
	buf.Fill(buf.Bounds(), blackCell)

	// Render each streak
	for _, streak := range m.streaks {
		if streak.startDelay > 0 {
			continue
		}

		// Skip if streak would be off the right edge (need 2 cols for char)
		if streak.xPos+2 > m.width {
			continue
		}

		for y := 0; y < m.height; y++ {
			distFromHead := streak.head - y

			// Skip if this position is not in the visible trail
			if distFromHead < 0 || distFromHead >= streak.length {
				continue
			}

			// Get the character for this position
			charIdx := y % len(streak.chars)
			char := streak.chars[charIdx]

			// Calculate color based on distance from head
			style := m.getCharStyle(distFromHead, streak.length)

			// Write the character at the streak's x position
			// Always occupy 2 cells in the buffer
			charWidth := runewidth.RuneWidth(char)
			if charWidth == 1 {
				// Single-width char + space = 2 cells, 2 terminal columns
				buf.SetString(streak.xPos, y, string(char)+" ", style)
			} else {
				// Wide char takes 2 terminal columns but only 1 buffer cell
				// Put null rune (0) in next cell as placeholder - renderRow will skip it
				buf.SetCells(streak.xPos, y, []teapot.Cell{
					{Rune: char, Style: style},
					{Rune: 0, Style: style},
				})
			}
		}
	}
}

// getCharStyle returns the style for a character based on its position in the trail
func (m *Screensaver) getCharStyle(distFromHead, length int) lipgloss.Style {
	// Head character is bright white/green
	if distFromHead == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")). // Bright white
			Bold(true).
			Background(lipgloss.Color("0"))
	}

	// Calculate brightness based on distance from head
	// Closer to head = brighter green
	ratio := float64(distFromHead) / float64(length)

	var colorCode string
	if ratio < 0.2 {
		colorCode = "46" // Bright green
	} else if ratio < 0.4 {
		colorCode = "40" // Green
	} else if ratio < 0.6 {
		colorCode = "34" // Dark green
	} else if ratio < 0.8 {
		colorCode = "28" // Darker green
	} else {
		colorCode = "22" // Very dark green
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(colorCode)).
		Background(lipgloss.Color("0"))
}

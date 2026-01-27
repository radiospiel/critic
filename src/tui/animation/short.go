package animation

import "time"

// ShortType identifies different short (multi-character) animation styles.
type ShortType int

const (
	Wave ShortType = iota
	ProgressBar
	Snake
	Pulse
	Scan
	Bounce
	Fire
	Matrix
	Equalizer
	Loading
	Ripple
	Knight
)

// shortAnimations contains all available short animation definitions.
var shortAnimations = map[ShortType]Animation{
	Wave: {
		Frames: []string{
			"▁▂▃▄▅▆▇█▇▆▅▄",
			"▂▃▄▅▆▇█▇▆▅▄▃",
			"▃▄▅▆▇█▇▆▅▄▃▂",
			"▄▅▆▇█▇▆▅▄▃▂▁",
			"▅▆▇█▇▆▅▄▃▂▁▂",
			"▆▇█▇▆▅▄▃▂▁▂▃",
			"▇█▇▆▅▄▃▂▁▂▃▄",
			"█▇▆▅▄▃▂▁▂▃▄▅",
			"▇▆▅▄▃▂▁▂▃▄▅▆",
			"▆▅▄▃▂▁▂▃▄▅▆▇",
			"▅▄▃▂▁▂▃▄▅▆▇█",
			"▄▃▂▁▂▃▄▅▆▇█▇",
		},
		Colors:  []string{"#61AFEF"},
		Speed:   60 * time.Millisecond,
		Colored: true,
	},
	ProgressBar: {
		Frames: []string{
			"█▁▁▁▁▁▁▁▁▁▁▁",
			"██▁▁▁▁▁▁▁▁▁▁",
			"███▁▁▁▁▁▁▁▁▁",
			"████▁▁▁▁▁▁▁▁",
			"█████▁▁▁▁▁▁▁",
			"██████▁▁▁▁▁▁",
			"███████▁▁▁▁▁",
			"████████▁▁▁▁",
			"█████████▁▁▁",
			"██████████▁▁",
			"███████████▁",
			"████████████",
			"▁███████████",
			"▁▁██████████",
			"▁▁▁█████████",
			"▁▁▁▁████████",
			"▁▁▁▁▁███████",
			"▁▁▁▁▁▁██████",
			"▁▁▁▁▁▁▁█████",
			"▁▁▁▁▁▁▁▁████",
			"▁▁▁▁▁▁▁▁▁███",
			"▁▁▁▁▁▁▁▁▁▁██",
			"▁▁▁▁▁▁▁▁▁▁▁█",
		},
		Colors:  []string{"#98C379"},
		Speed:   50 * time.Millisecond,
		Colored: true,
	},
	Snake: {
		Frames: []string{
			"●○○○○○○○○○○○",
			"○●○○○○○○○○○○",
			"○○●○○○○○○○○○",
			"○○○●○○○○○○○○",
			"○○○○●○○○○○○○",
			"○○○○○●○○○○○○",
			"○○○○○○●○○○○○",
			"○○○○○○○●○○○○",
			"○○○○○○○○●○○○",
			"○○○○○○○○○●○○",
			"○○○○○○○○○○●○",
			"○○○○○○○○○○○●",
			"○○○○○○○○○○●○",
			"○○○○○○○○○●○○",
			"○○○○○○○○●○○○",
			"○○○○○○○●○○○○",
			"○○○○○○●○○○○○",
			"○○○○○●○○○○○○",
			"○○○○●○○○○○○○",
			"○○○●○○○○○○○○",
			"○○●○○○○○○○○○",
			"○●○○○○○○○○○○",
		},
		Colors:  []string{"#E5C07B"},
		Speed:   70 * time.Millisecond,
		Colored: true,
	},
	Pulse: {
		Frames: []string{
			"▁▁▁▁▁▁▁▁▁▁▁▁",
			"▂▂▂▂▂▂▂▂▂▂▂▂",
			"▃▃▃▃▃▃▃▃▃▃▃▃",
			"▄▄▄▄▄▄▄▄▄▄▄▄",
			"▅▅▅▅▅▅▅▅▅▅▅▅",
			"▆▆▆▆▆▆▆▆▆▆▆▆",
			"▇▇▇▇▇▇▇▇▇▇▇▇",
			"████████████",
			"▇▇▇▇▇▇▇▇▇▇▇▇",
			"▆▆▆▆▆▆▆▆▆▆▆▆",
			"▅▅▅▅▅▅▅▅▅▅▅▅",
			"▄▄▄▄▄▄▄▄▄▄▄▄",
			"▃▃▃▃▃▃▃▃▃▃▃▃",
			"▂▂▂▂▂▂▂▂▂▂▂▂",
		},
		Colors:  []string{"#C678DD"},
		Speed:   50 * time.Millisecond,
		Colored: true,
	},
	Scan: {
		Frames: []string{
			"█▁▁▁▁▁▁▁▁▁▁▁",
			"▁█▁▁▁▁▁▁▁▁▁▁",
			"▁▁█▁▁▁▁▁▁▁▁▁",
			"▁▁▁█▁▁▁▁▁▁▁▁",
			"▁▁▁▁█▁▁▁▁▁▁▁",
			"▁▁▁▁▁█▁▁▁▁▁▁",
			"▁▁▁▁▁▁█▁▁▁▁▁",
			"▁▁▁▁▁▁▁█▁▁▁▁",
			"▁▁▁▁▁▁▁▁█▁▁▁",
			"▁▁▁▁▁▁▁▁▁█▁▁",
			"▁▁▁▁▁▁▁▁▁▁█▁",
			"▁▁▁▁▁▁▁▁▁▁▁█",
			"▁▁▁▁▁▁▁▁▁▁█▁",
			"▁▁▁▁▁▁▁▁▁█▁▁",
			"▁▁▁▁▁▁▁▁█▁▁▁",
			"▁▁▁▁▁▁▁█▁▁▁▁",
			"▁▁▁▁▁▁█▁▁▁▁▁",
			"▁▁▁▁▁█▁▁▁▁▁▁",
			"▁▁▁▁█▁▁▁▁▁▁▁",
			"▁▁▁█▁▁▁▁▁▁▁▁",
			"▁▁█▁▁▁▁▁▁▁▁▁",
			"▁█▁▁▁▁▁▁▁▁▁▁",
		},
		Colors:  []string{"#E06C75"},
		Speed:   40 * time.Millisecond,
		Colored: true,
	},
	Bounce: {
		Frames: []string{
			"●▁▁▁▁▁▁▁▁▁▁▁",
			"▁●▁▁▁▁▁▁▁▁▁▁",
			"▁▁●▁▁▁▁▁▁▁▁▁",
			"▁▁▁●▁▁▁▁▁▁▁▁",
			"▁▁▁▁●▁▁▁▁▁▁▁",
			"▁▁▁▁▁●▁▁▁▁▁▁",
			"▁▁▁▁▁▁●▁▁▁▁▁",
			"▁▁▁▁▁▁▁●▁▁▁▁",
			"▁▁▁▁▁▁▁▁●▁▁▁",
			"▁▁▁▁▁▁▁▁▁●▁▁",
			"▁▁▁▁▁▁▁▁▁▁●▁",
			"▁▁▁▁▁▁▁▁▁▁▁●",
			"▁▁▁▁▁▁▁▁▁▁●▁",
			"▁▁▁▁▁▁▁▁▁●▁▁",
			"▁▁▁▁▁▁▁▁●▁▁▁",
			"▁▁▁▁▁▁▁●▁▁▁▁",
			"▁▁▁▁▁▁●▁▁▁▁▁",
			"▁▁▁▁▁●▁▁▁▁▁▁",
			"▁▁▁▁●▁▁▁▁▁▁▁",
			"▁▁▁●▁▁▁▁▁▁▁▁",
			"▁▁●▁▁▁▁▁▁▁▁▁",
			"▁●▁▁▁▁▁▁▁▁▁▁",
		},
		Colors:  []string{"#56B6C2"},
		Speed:   50 * time.Millisecond,
		Colored: true,
	},
	Fire: {
		Frames: []string{
			"▁▂▃▄▅▆▇█▇▆▅▄",
			"▂▁▄▃▆▅█▇▆▇▄▅",
			"▃▄▁▆▅▇█▆▇▆▅▄",
			"▄▃▆▁█▅▇▆▆▇▄▅",
			"▅▄▃▇▁█▆▇▇▆▅▄",
			"▆▅▄█▆▁▇▆▆▇▄▅",
			"▇▆▅▇█▇▁▆▇▆▅▄",
			"█▇▆▆▇▆█▁▆▇▄▅",
			"▇█▇▇▆▇▇█▁▆▅▄",
			"▆▇█▆▇▆▆▇█▁▄▅",
			"▅▆▇▇▆▇▇▆▇█▁▄",
			"▄▅▆▆▇▆▆▇▆▇█▁",
		},
		Colors:  []string{"#E06C75", "#E5C07B", "#E06C75"},
		Speed:   60 * time.Millisecond,
		Colored: true,
	},
	Matrix: {
		Frames: []string{
			"█▇▆▅▄▃▂▁▁▂▃▄",
			"▇█▇▆▅▄▃▂▁▁▂▃",
			"▆▇█▇▆▅▄▃▂▁▁▂",
			"▅▆▇█▇▆▅▄▃▂▁▁",
			"▄▅▆▇█▇▆▅▄▃▂▁",
			"▃▄▅▆▇█▇▆▅▄▃▂",
			"▂▃▄▅▆▇█▇▆▅▄▃",
			"▁▂▃▄▅▆▇█▇▆▅▄",
			"▁▁▂▃▄▅▆▇█▇▆▅",
			"▂▁▁▂▃▄▅▆▇█▇▆",
			"▃▂▁▁▂▃▄▅▆▇█▇",
			"▄▃▂▁▁▂▃▄▅▆▇█",
		},
		Colors:  []string{"#98C379"},
		Speed:   70 * time.Millisecond,
		Colored: true,
	},
	Equalizer: {
		Frames: []string{
			"▃▅▂▇▄█▅▃▆▂▇▄",
			"▅▃▇▄█▅▃▆▂▇▄▂",
			"▃▇▄█▅▃▆▂▇▄▂▅",
			"▇▄█▅▃▆▂▇▄▂▅▃",
			"▄█▅▃▆▂▇▄▂▅▃▇",
			"█▅▃▆▂▇▄▂▅▃▇▄",
			"▅▃▆▂▇▄▂▅▃▇▄█",
			"▃▆▂▇▄▂▅▃▇▄█▅",
			"▆▂▇▄▂▅▃▇▄█▅▃",
			"▂▇▄▂▅▃▇▄█▅▃▆",
			"▇▄▂▅▃▇▄█▅▃▆▂",
			"▄▂▅▃▇▄█▅▃▆▂▇",
		},
		Colors:  []string{"#61AFEF", "#56B6C2", "#98C379"},
		Speed:   80 * time.Millisecond,
		Colored: true,
	},
	Loading: {
		Frames: []string{
			"[          ]",
			"[=         ]",
			"[==        ]",
			"[===       ]",
			"[====      ]",
			"[=====     ]",
			"[======    ]",
			"[=======   ]",
			"[========  ]",
			"[========= ]",
			"[==========]",
			"[ =========]",
			"[  ========]",
			"[   =======]",
			"[    ======]",
			"[     =====]",
			"[      ====]",
			"[       ===]",
			"[        ==]",
			"[         =]",
		},
		Colors:  []string{"#C678DD"},
		Speed:   60 * time.Millisecond,
		Colored: true,
	},
	Ripple: {
		Frames: []string{
			"▁▁▁▁▁█▁▁▁▁▁▁",
			"▁▁▁▁▃█▃▁▁▁▁▁",
			"▁▁▁▂▅█▅▂▁▁▁▁",
			"▁▁▂▄▆█▆▄▂▁▁▁",
			"▁▂▃▅▇█▇▅▃▂▁▁",
			"▂▃▄▆██▆▄▃▂▁▁",
			"▃▄▅▇██▇▅▄▃▂▁",
			"▄▅▆███▆▅▄▃▂▁",
			"▅▆▇███▇▆▅▄▃▂",
			"▆▇████▇▆▅▄▃▂",
			"▇█████▇▆▅▄▃▂",
			"██████▇▆▅▄▃▂",
			"█████▇▆▅▄▃▂▁",
			"████▇▆▅▄▃▂▁▁",
			"███▇▆▅▄▃▂▁▁▁",
			"██▇▆▅▄▃▂▁▁▁▁",
			"█▇▆▅▄▃▂▁▁▁▁▁",
			"▇▆▅▄▃▂▁▁▁▁▁▁",
			"▆▅▄▃▂▁▁▁▁▁▁▁",
			"▅▄▃▂▁▁▁▁▁▁▁▁",
			"▄▃▂▁▁▁▁▁▁▁▁▁",
			"▃▂▁▁▁▁▁▁▁▁▁▁",
			"▂▁▁▁▁▁▁▁▁▁▁▁",
		},
		Colors:  []string{"#56B6C2"},
		Speed:   50 * time.Millisecond,
		Colored: true,
	},
	Knight: {
		Frames: []string{
			"♞···········",
			"·♞··········",
			"··♞·········",
			"···♞········",
			"····♞·······",
			"·····♞······",
			"······♞·····",
			"·······♞····",
			"········♞···",
			"·········♞··",
			"··········♞·",
			"···········♞",
		},
		Colors:  []string{"#E5C07B"},
		Speed:   80 * time.Millisecond,
		Colored: true,
	},
}

// Short type names (indexed by ShortType)
var shortTypeNames = []string{
	"Wave",
	"Progress Bar",
	"Snake",
	"Pulse",
	"Scan",
	"Bounce",
	"Fire",
	"Matrix",
	"Equalizer",
	"Loading",
	"Ripple",
	"Knight Rider",
}

// Name returns the human-readable name of the short animation type.
func (t ShortType) Name() string {
	if int(t) < len(shortTypeNames) {
		return shortTypeNames[t]
	}
	return "Unknown"
}

// GetShort returns a copy of the Animation for the given short type.
func GetShort(t ShortType) Animation {
	return shortAnimations[t]
}

// NewShortAnimation returns a new Animation configured for the given short type
// with the specified color mode and speed factor.
// Speed factor: 1.0 = normal, <1.0 = faster, >1.0 = slower.
func NewShortAnimation(t ShortType, colored bool, speedFactor float64) *Animation {
	base := shortAnimations[t]
	return &Animation{
		Frames:  base.Frames,
		Colors:  base.Colors,
		Speed:   time.Duration(float64(base.Speed) * speedFactor),
		Colored: colored,
		Frame:   0,
	}
}


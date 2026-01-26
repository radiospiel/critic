package config

// ShiftArrowJumpSize is the number of lines to jump when using Shift+Up/Down
const ShiftArrowJumpSize = 10

// DefaultFileExtensions lists the file extensions that are included by default
// when scanning for files to diff. This includes common code file types and
// documentation formats.
var DefaultFileExtensions = []string{
	// Go
	"go",

	// Rust
	"rs",

	// C/C++
	"c", "cpp", "cc", "cxx", "c++",
	"h", "hpp", "hh", "hxx", "h++",

	// JavaScript/TypeScript
	"js", "mjs", "cjs",
	"ts", "mts", "cts",
	"jsx", "tsx",

	// Python
	"py", "pyw", "pyi",

	// Ruby
	"rb", "rake", "gemspec",

	// Java/Kotlin
	"java",
	"kt", "kts",

	// C#
	"cs",

	// PHP
	"php",

	// Shell scripts
	"sh", "bash", "zsh", "fish",

	// Perl
	"pl", "pm",

	// Lua
	"lua",

	// Elixir
	"ex", "exs",

	// Erlang
	"erl", "hrl",

	// Haskell
	"hs",

	// Swift
	"swift",

	// Scala
	"scala",

	// R
	"r", "R",

	// Julia
	"jl",

	// Documentation
	"md", "markdown",
	"rst",
	"txt",

	// Configuration
	"yml", "yaml",
	"toml",
	"json",
	"xml",
	"ini",
	"conf", "config",

	// Web
	"html", "htm",
	"css", "scss", "sass", "less",
	"vue",
	"svelte",

	// SQL
	"sql",

	// Protobuf
	"proto",

	// Makefiles and scripts
	"mk",
	"cmake",
}

// HasExtension checks if a file path has one of the given extensions.
// If extensions is nil or empty, returns true (no filtering).
func HasExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}

	// Extract extension from path
	lastDot := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			lastDot = i
			break
		}
		if path[i] == '/' {
			break // No extension found
		}
	}

	if lastDot == -1 || lastDot == len(path)-1 {
		return false // No extension
	}

	ext := path[lastDot+1:]

	// Check if extension matches
	for _, allowed := range extensions {
		if ext == allowed {
			return true
		}
	}

	return false
}

# Add Syntax Highlighting for Well-Known Languages

Started: 2026-01-28 10:51:05
Ended: 2026-01-28 10:55:00
Complexity: Simple

## Summary
Enhance syntax highlighting in the webui by adding support for more well-known programming languages.

## Task Type
**Feature** - Adding new functionality (extended language support)

## Progress

- [x] Explore codebase structure
- [x] Review existing implementation
- [x] Extend language mappings
- [x] Verify build compiles
- [x] Commit and push

## Findings

The webui already had syntax highlighting via highlight.js, but the language mapping was limited to ~24 extensions.

Extended to ~80+ file extensions covering:
- JavaScript/TypeScript (including .mjs, .cjs, .mts, .cts)
- Python, Go, Rust, Ruby (with Ruby config files like Gemfile, Rakefile)
- Java/JVM languages (Kotlin, Scala, Groovy, Clojure)
- C/C++/Objective-C with all common extensions
- C#, F#
- Web files (HTML, XML, Vue, Svelte, Astro)
- CSS/SCSS/Less/Stylus
- Shell scripts (Bash, Zsh, Fish, PowerShell, DOS)
- Data formats (JSON, YAML, TOML, INI)
- Database (SQL, PgSQL)
- Other languages (PHP, Swift, Perl, Lua, R, Julia, Elixir, Erlang, Haskell, OCaml, Elm, Dart, Zig, Nim, D)
- Lisp family (Lisp, Scheme)
- Config/DevOps (Proto, GraphQL, Terraform/HCL, Nix)
- Assembly
- Special files (Dockerfile, Makefile, .env files, .gitignore)

Also added case-insensitive extension matching.

## Obstacles

None - straightforward extension of existing functionality.

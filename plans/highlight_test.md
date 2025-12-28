# Fix tab

The hightlighter has a bug. The first tab in the highlighted code
renders 1 to 4 spaces, depending on the position of the source pane.

Fix this by

- replacing tabs to spaces before highlighting.
- the number of spaces per tab is language dependant. Add a config which translates
  - ruby: 2 spaces 
  - go: 4 spacecs
  - default: 4 spaces
- the resulting string should not have spaces
- add realworld test cases. Take the "test_highlight.go" file as inspiration, build test cases that read a specific source file and return a highlighted 
  string.
- add test cases with emoji, german umlauts, and other non ascii cases
- as a result the expandTabsInANSI should no longer be necessary and can be dropped.



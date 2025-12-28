Introduce the following:

- an interface critic/DiffState, which exposed the following with some functions
  - list of names of changed files, with their diff state (created/deleted/changed)
  - DiffDetails: list of hunks per filename, content of original and current file
  - a function callback (onChange) which emits old and new diffdetails. If a file goes from changed to unchanged (i.e. reverted to the original) emit an event as well.
- The current implementation is wrapped behind a GitDiffState(..paths..) -> &DiffState function

- an interface ui/ViewState, which exposes
  - the name of the currently selected file, if any
  - the hunk that the active line is currently in, and the position of the active line within the hunk
  
  


  
  
Change and extend the use of CLI args:

- we support the following syntaxes
  
  critic --extensions=c,rb,go base1,base2,base3..current -- path1 path2 path3

- base1,etc. are "the starting points" of the diff. This is what the user can switch thru iusing the <m> (mode) key. Change that to <b> (for "base") Note that you need to show the active "base" in the UI. Can be git tags, commits, and branches, and defaults to the current state on disk, including ignored and non-merged files. When running the diff, we are not diffing from the literal, e.g., "master", but from the merge point between "current" and the base.
- current is the diff target. Can be git tags, commits, and branches, and defaults to the current state on disk, including ignored and non-merged files.
- path1 path2 ... are paths to show the diff for. Default to "."
- extensions is a list of file extensions. The default is configured in the config.go package, and the default lists a list of typical code-related file extensions, and also ".md" and ".markdown"

Most parts of these args are optional. The following are the defaults:

-  starting points:
	- the merge base against main (if that exists) or master. 
	- if we are on a branch, the origin of the current branch (i.e. origin/name-of-branch)
	- if we are on a branch, the last committed version
- current:
    - defaults to the current state on disk, including ignored and non-merged files.
- paths: "."

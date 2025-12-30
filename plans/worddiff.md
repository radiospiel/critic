use git diff --word-diff 

e.g.
	git diff --word-diff master -- internal/git/watcher.go
	
and highlight deleted words by strike trhu w/red background, and new workds by grren background. I.e. don't render both lines. Can we use --color-words?



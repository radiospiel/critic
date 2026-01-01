- critic comments are stored in files next to the commented file.
- the file name is <original-file>.critic.md. <original-file> includes the file extension
- the file consist of the <original-file>'s content, with critic blocks embedded
- a critic block looks like this

		--- CRITIC 8 lines ------------------------------
		eight lines of comment
		.
		.
		.
		.
		.
		.
		.
		--- CRITIC END ------------------------------ 

- when extracting a critic block we only rely on the --- CRITIC opening fence and the number of lines in there, and we validate that with then --- CRITIC end fence.
- a user can add or modify a critic block from the details pane pressing return. They can save pressing ctrl save, save and exit with ctrl x, and abort using escape.
- When saving the comments, we generate two files:
  - a <original-file>.critic.original file, which is just a copy of the file in the diff pane without any extensions, and
  - a <original-file>.critic.md file, which is a copy of the file with comment fences in the correct plance
- when we receive an update event on the original file, we do the following:
  - calculate the diff between <original-file>.critic.original and <updated-original-file>
  - apply that diff on <original-file>.critic.md 
  - test that we can then parse the updated <original-file>.critic.md, that it contains all lines in <updated-original-file>, and that all critic blocks are consistent, and are already present in the  <original-file>.critic.md 
  
  
  
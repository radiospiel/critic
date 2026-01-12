#!/bin/bash
set -eu -o pipefail
HERE=$(dirname "${BASH_SOURCE[0]}")
cd "$HERE"

rm -rf repo
mkdir repo
cd repo
git init
for file in ../v*.txt ; do
	cp $file data.txt
	version=$(basename $file | sed 's-\..*--')
	git add data.txt
	git commit -m "version $version" 
	git tag $version
done

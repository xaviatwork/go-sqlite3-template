#!/usr/bin/env bash

grep --quiet --fixed-strings --line-regexp 'source .devcontainer/git-completion.bash' ~/.bashrc || echo 'source .devcontainer/git-completion.bash' >> ~/.bashrc

sudo apt update && sudo apt install sqlite3 --yes
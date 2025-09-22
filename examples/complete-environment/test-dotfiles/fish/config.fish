# Basic fish configuration (non-template version)
set -g fish_greeting ""

# Environment variables
set -gx EDITOR vim
set -gx LANG en_US.UTF-8

# Path modifications
fish_add_path /usr/local/bin
fish_add_path ~/.local/bin

# Basic aliases
alias ll "ls -la"
alias la "ls -A"
alias l "ls -CF"
alias grep "grep --color=auto"

# Git aliases
alias gs "git status"
alias ga "git add"
alias gc "git commit"
alias gp "git push"
alias gl "git log --oneline"

# Functions
function mkcd
    mkdir -p $argv[1]
    cd $argv[1]
end

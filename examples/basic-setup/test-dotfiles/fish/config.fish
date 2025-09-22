# Fish shell configuration
set -g fish_greeting ""

# Environment variables
set -gx EDITOR vim
set -gx LANG en_US.UTF-8

# Path modifications
fish_add_path /usr/local/bin
fish_add_path ~/.local/bin

# Aliases
alias ll "ls -la"
alias la "ls -A"
alias l "ls -CF"

# Functions
function mkcd
    mkdir -p $argv[1]
    cd $argv[1]
end

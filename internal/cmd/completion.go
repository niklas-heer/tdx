package cmd

import (
	"fmt"
	"io"
)

// WriteCompletion emits static shell code. It never opens task files, queries
// history, or evaluates document text as shell syntax.
func WriteCompletion(out io.Writer, shell string) error {
	var script string
	switch shell {
	case "bash":
		script = bashCompletion
	case "zsh":
		script = zshCompletion
	case "fish":
		script = fishCompletion
	default:
		return fmt.Errorf("unsupported completion shell %q; use bash, zsh, or fish", shell)
	}
	_, err := io.WriteString(out, script)
	return err
}

const bashCompletion = `# Load with: source <(tdx completion bash)
_tdx_complete() {
    local cur prev word choices i
    cur=${COMP_WORDS[COMP_CWORD]}
    prev=${COMP_WORDS[COMP_CWORD-1]}
    COMPREPLY=()
    for ((i=1; i<COMP_CWORD; i++)); do
        [[ ${COMP_WORDS[i]} != -- ]] || return
    done
    case "$prev" in
        --file|-f) return ;;
        --status) choices='all open done' ;;
        --due) choices='all none overdue today week' ;;
        --priority) choices='0 1 2 3 4' ;;
        --tag|--section|--if-revision|--max-visible|-m) return ;;
        completion) choices='bash zsh fish' ;;
        *) choices='list add toggle done undone edit delete revision completion last recent help --file --read-only --manual-save --show-headings --max-visible --json --with-revision --if-revision --status --tag --priority --due --section --help --version' ;;
    esac
    while IFS= read -r word; do
        COMPREPLY+=("$word")
    done < <(compgen -W "$choices" -- "$cur")
}
complete -o default -F _tdx_complete tdx
`

const zshCompletion = `# Load after compinit: source <(tdx completion zsh)
_tdx_complete() {
    local curcontext="$curcontext" state line
    _arguments -s \
      '(-f --file)'{-f,--file}'[Task file]:file:_files' \
      '(-r --read-only --manual-save)'{-r,--read-only,--manual-save}'[Manual save in TUI; reject CLI writes]' \
      '--show-headings[Display headings]' \
      '(-m --max-visible)'{-m,--max-visible}'[Maximum visible tasks]:count:' \
      '--json[JSON task array]' '--with-revision[Versioned JSON snapshot]' \
      '--if-revision[Require matching document revision]:revision:' \
      '--status[Completion filter]:status:(all open done)' \
      '*--tag[Require tag]:tag:' \
      '--priority[Priority filter]:priority:(0 1 2 3 4)' \
      '--due[Due date filter]:due:(all none overdue today week)' \
      '--section[Heading and its subsections]:heading:' \
      '(-h --help)'{-h,--help}'[Show help]' \
      '(-v --version)'{-v,--version}'[Show version]' \
      '*:argument:->arguments'
    if [[ $state == arguments ]]; then
        if [[ ${words[CURRENT-1]} == completion ]]; then
            compadd bash zsh fish
        else
            compadd list add toggle done undone edit delete revision completion last recent help
            _files
        fi
    fi
}
compdef _tdx_complete tdx
`

const fishCompletion = `# Save to ~/.config/fish/completions/tdx.fish
complete -c tdx -n __fish_use_subcommand -a 'list add toggle done undone edit delete revision completion last recent help'
complete -c tdx -n '__fish_seen_subcommand_from completion' -f -a 'bash zsh fish'
complete -c tdx -s f -l file -r -d 'Task file'
complete -c tdx -s r -l read-only -d 'Manual save in TUI; reject CLI writes'
complete -c tdx -l manual-save -d 'Alias for read-only'
complete -c tdx -l show-headings -d 'Display headings'
complete -c tdx -s m -l max-visible -x -d 'Maximum visible tasks'
complete -c tdx -l json -d 'JSON task array'
complete -c tdx -l with-revision -d 'Versioned JSON snapshot'
complete -c tdx -l if-revision -x -d 'Require matching revision'
complete -c tdx -l status -x -a 'all open done'
complete -c tdx -l tag -x -d 'Require tag'
complete -c tdx -l priority -x -a '0 1 2 3 4'
complete -c tdx -l due -x -a 'all none overdue today week'
complete -c tdx -l section -x -d 'Heading and its subsections'
complete -c tdx -s h -l help -d 'Show help'
complete -c tdx -s v -l version -d 'Show version'
`

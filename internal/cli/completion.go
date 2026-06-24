package cli

const bashCompletion = `# bash completion for tmc
_tmc() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "init start stop list status doctor dry-run completion version help" -- "${cur}") )
    return 0
  fi
  case "${prev}" in
    completion)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- "${cur}") )
      return 0
      ;;
  esac
}
complete -F _tmc tmc
`

const zshCompletion = `#compdef tmc
_tmc() {
  local -a commands
  commands=(
    'init:generate a starter manifest'
    'start:start a workspace'
    'stop:stop a session'
    'list:list tmux sessions'
    'status:show session status'
    'doctor:run dependency and manifest checks'
    'dry-run:print planned actions'
    'completion:emit shell completion'
    'version:show build version'
    'help:show help'
  )
  _describe 'command' commands
}
compdef _tmc tmc
`

const fishCompletion = `complete -c tmc -f
complete -c tmc -n "__fish_use_subcommand" -a "init start stop list status doctor dry-run completion version help"
complete -c tmc -n "__fish_seen_subcommand_from completion" -a "bash zsh fish"
`

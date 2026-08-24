package completion

import (
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func renderBash(commands []*ffcli.Command) string {
	var result strings.Builder
	fmt.Fprintln(&result, "# gplay bash completion script")
	fmt.Fprintln(&result, "# Generated from the gplay command catalog")
	fmt.Fprintln(&result, "_gplay_completions() {")
	fmt.Fprintln(&result, "    local cur prev words cword")
	fmt.Fprintln(&result, "    _init_completion || return")
	fmt.Fprintf(&result, "    local commands=%q\n", strings.Join(commandNames(commands), " "))
	fmt.Fprintln(&result, "    if [[ ${cword} -eq 1 ]]; then")
	fmt.Fprintln(&result, "        COMPREPLY=($(compgen -W \"${commands}\" -- \"${cur}\"))")
	fmt.Fprintln(&result, "        return")
	fmt.Fprintln(&result, "    fi")
	fmt.Fprintln(&result, "    if [[ ${cword} -eq 2 ]]; then")
	fmt.Fprintln(&result, "        case \"${words[1]}\" in")
	for _, command := range visibleCommands(commands) {
		if len(visibleCommands(command.Subcommands)) == 0 {
			continue
		}
		fmt.Fprintf(&result, "            %s) COMPREPLY=($(compgen -W %q -- \"${cur}\")) ;;\n",
			command.Name, strings.Join(commandNames(command.Subcommands), " "))
	}
	fmt.Fprintln(&result, "        esac")
	fmt.Fprintln(&result, "    fi")
	fmt.Fprintln(&result, "}")
	fmt.Fprintln(&result, "complete -F _gplay_completions gplay")
	return result.String()
}

func renderZsh(commands []*ffcli.Command) string {
	var result strings.Builder
	fmt.Fprintln(&result, "#compdef gplay")
	fmt.Fprintln(&result, "# Generated from the gplay command catalog")
	fmt.Fprintln(&result, "_gplay() {")
	fmt.Fprintln(&result, "  local -a commands")
	fmt.Fprintln(&result, "  commands=(")
	for _, command := range visibleCommands(commands) {
		fmt.Fprintf(&result, "    '%s:%s'\n", zshQuote(command.Name), zshQuote(command.ShortHelp))
	}
	fmt.Fprintln(&result, "  )")
	fmt.Fprintln(&result, "  if (( CURRENT == 2 )); then")
	fmt.Fprintln(&result, "    _describe 'command' commands")
	fmt.Fprintln(&result, "    return")
	fmt.Fprintln(&result, "  fi")
	fmt.Fprintln(&result, "  case $words[2] in")
	for _, command := range visibleCommands(commands) {
		subcommands := visibleCommands(command.Subcommands)
		if len(subcommands) == 0 {
			continue
		}
		fmt.Fprintf(&result, "    %s)\n", command.Name)
		fmt.Fprintln(&result, "      local -a subcommands")
		fmt.Fprintln(&result, "      subcommands=(")
		for _, subcommand := range subcommands {
			fmt.Fprintf(&result, "        '%s:%s'\n", zshQuote(subcommand.Name), zshQuote(subcommand.ShortHelp))
		}
		fmt.Fprintln(&result, "      )")
		fmt.Fprintln(&result, "      _describe 'subcommand' subcommands")
		fmt.Fprintln(&result, "      ;;")
	}
	fmt.Fprintln(&result, "  esac")
	fmt.Fprintln(&result, "}")
	fmt.Fprintln(&result, "compdef _gplay gplay")
	return result.String()
}

func renderFish(commands []*ffcli.Command) string {
	var result strings.Builder
	fmt.Fprintln(&result, "# gplay fish completion script")
	fmt.Fprintln(&result, "# Generated from the gplay command catalog")
	for _, command := range visibleCommands(commands) {
		fmt.Fprintf(&result, "complete -c gplay -n '__fish_use_subcommand' -a '%s' -d '%s'\n",
			fishQuote(command.Name), fishQuote(command.ShortHelp))
		for _, subcommand := range visibleCommands(command.Subcommands) {
			fmt.Fprintf(&result, "complete -c gplay -n '__fish_seen_subcommand_from %s' -a '%s' -d '%s'\n",
				fishQuote(command.Name), fishQuote(subcommand.Name), fishQuote(subcommand.ShortHelp))
		}
	}
	return result.String()
}

func renderPowerShell(commands []*ffcli.Command) string {
	var result strings.Builder
	fmt.Fprintln(&result, "# gplay PowerShell completion script")
	fmt.Fprintln(&result, "# Generated from the gplay command catalog")
	fmt.Fprintln(&result, "$gplayCommands = @(")
	for _, command := range visibleCommands(commands) {
		fmt.Fprintf(&result, "  '%s'\n", powerShellQuote(command.Name))
	}
	fmt.Fprintln(&result, ")")
	fmt.Fprintln(&result, "$gplaySubcommands = @{")
	for _, command := range visibleCommands(commands) {
		subcommands := commandNames(command.Subcommands)
		if len(subcommands) == 0 {
			continue
		}
		quoted := make([]string, 0, len(subcommands))
		for _, subcommand := range subcommands {
			quoted = append(quoted, "'"+powerShellQuote(subcommand)+"'")
		}
		fmt.Fprintf(&result, "  '%s' = @(%s)\n", powerShellQuote(command.Name), strings.Join(quoted, ", "))
	}
	fmt.Fprintln(&result, "}")
	fmt.Fprintln(&result, "Register-ArgumentCompleter -Native -CommandName gplay -ScriptBlock {")
	fmt.Fprintln(&result, "  param($wordToComplete, $commandAst, $cursorPosition)")
	fmt.Fprintln(&result, "  $tokens = @($commandAst.CommandElements | ForEach-Object { $_.Extent.Text })")
	fmt.Fprintln(&result, "  $candidates = if ($tokens.Count -le 2) { $gplayCommands } else { $gplaySubcommands[$tokens[1]] }")
	fmt.Fprintln(&result, "  $candidates | Where-Object { $_ -like \"$wordToComplete*\" } | ForEach-Object {")
	fmt.Fprintln(&result, "    [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)")
	fmt.Fprintln(&result, "  }")
	fmt.Fprintln(&result, "}")
	return result.String()
}

func commandNames(commands []*ffcli.Command) []string {
	commands = visibleCommands(commands)
	result := make([]string, 0, len(commands))
	for _, command := range commands {
		result = append(result, command.Name)
	}
	return result
}

func visibleCommands(commands []*ffcli.Command) []*ffcli.Command {
	result := make([]*ffcli.Command, 0, len(commands))
	for _, command := range commands {
		if command == nil || strings.HasPrefix(command.ShortHelp, "DEPRECATED:") {
			continue
		}
		result = append(result, command)
	}
	return result
}

func zshQuote(value string) string {
	value = strings.ReplaceAll(value, "'", "'\\''")
	return strings.ReplaceAll(value, ":", "\\:")
}

func fishQuote(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

func powerShellQuote(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

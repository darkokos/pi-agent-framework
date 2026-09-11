#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
BIN="$(pwd)/bin/pi-agent-lab"

if [[ ! -x "$BIN" ]]; then
	echo "bin/pi-agent-lab not found; run scripts/build.sh first" >&2
	exit 1
fi

detect_shell() {
	basename "${SHELL:-}"
}

# Appends line to file under a marker comment, unless it's already there -
# safe to run this script repeatedly.
append_once() {
	local file="$1" line="$2"
	mkdir -p "$(dirname "$file")"
	touch "$file"
	if grep -qF "$line" "$file"; then
		echo "already present in $file"
		return 0
	fi
	printf '\n# pi-agent-lab shell completion\n%s\n' "$line" >>"$file"
	echo "added to $file:"
	echo "    $line"
}

install_fish() {
	local dir="$HOME/.config/fish/completions"
	mkdir -p "$dir"
	"$BIN" completion fish >"$dir/pi-agent-lab.fish"
	echo "installed fish completions to $dir/pi-agent-lab.fish"
}

# ~/.oh-my-zsh (outside its own custom/ dir) is the framework's own
# git-managed tree; a file dropped there gets wiped on the next update.
is_framework_managed_zsh_dir() {
	[[ "$1" == *"/.oh-my-zsh/"* && "$1" != *"/.oh-my-zsh/custom/"* ]]
}

install_zsh() {
	local dir
	local -a candidates=()
	while IFS= read -r dir; do
		[[ -z "$dir" ]] && continue
		is_framework_managed_zsh_dir "$dir" && continue
		candidates+=("$dir")
	done < <(zsh -i -c 'print -l $fpath' 2>/dev/null)

	# Prefer a directory whose name says "completions" over some other
	# writable fpath entry (e.g. a framework's generic "functions" dir).
	local pass
	for pass in completions any; do
		for dir in "${candidates[@]+"${candidates[@]}"}"; do
			[[ "$pass" == "completions" && "$dir" != *completions* ]] && continue
			if mkdir -p "$dir" 2>/dev/null && "$BIN" completion zsh >"$dir/_pi-agent-lab" 2>/dev/null; then
				echo "installed zsh completions to $dir/_pi-agent-lab"
				return 0
			fi
		done
	done

	echo "could not find a writable directory on your zsh \$fpath"
	append_once "$HOME/.zshrc" 'source <(pi-agent-lab completion zsh)'
}

install_bash() {
	local versinfo user_dir xdg_data dir
	versinfo=$(bash -i -c 'echo "${BASH_COMPLETION_VERSINFO:-}"' 2>/dev/null || true)
	if [[ -z "$versinfo" ]]; then
		echo "the bash-completion framework isn't active in your shell"
		append_once "$HOME/.bashrc" 'source <(pi-agent-lab completion bash)'
		return 0
	fi

	user_dir=$(bash -i -c 'echo "${BASH_COMPLETION_USER_DIR:-}"' 2>/dev/null || true)
	if [[ -n "$user_dir" ]]; then
		dir="$user_dir/completions"
	else
		xdg_data=$(bash -i -c 'echo "${XDG_DATA_HOME:-}"' 2>/dev/null || true)
		dir="${xdg_data:-$HOME/.local/share}/bash-completion/completions"
	fi

	mkdir -p "$dir"
	"$BIN" completion bash >"$dir/pi-agent-lab"
	echo "installed bash completions to $dir/pi-agent-lab"
}

install_powershell() {
	local profile line='pi-agent-lab completion powershell | Out-String | Invoke-Expression'
	if command -v pwsh >/dev/null 2>&1; then
		profile=$(pwsh -NoProfile -Command 'Write-Output $PROFILE' 2>/dev/null || true)
	fi
	profile="${profile:-$HOME/.config/powershell/Microsoft.PowerShell_profile.ps1}"

	echo "PowerShell has no auto-loaded completions directory"
	append_once "$profile" "$line"
}

shell="${1:-$(detect_shell)}"
case "$shell" in
bash) install_bash ;;
zsh) install_zsh ;;
fish) install_fish ;;
powershell | pwsh) install_powershell ;;
*)
	echo "unknown or undetected shell '$shell'; pass one explicitly: bash|zsh|fish|powershell" >&2
	exit 1
	;;
esac

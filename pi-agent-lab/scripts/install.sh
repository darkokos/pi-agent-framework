#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."
BIN="$(pwd)/bin/pi-agent-lab"

./scripts/build.sh

./scripts/install-completions.sh || echo "warning: shell completion install failed; run scripts/install-completions.sh manually" >&2

on_path() {
	case ":$PATH:" in
	*":$1:"*) return 0 ;;
	*) return 1 ;;
	esac
}

writable_dir() {
	if [[ -d "$1" ]]; then
		[[ -w "$1" ]]
		return
	fi
	mkdir -p "$1" 2>/dev/null
}

link_into() {
	local dir="$1"
	if ln -sf "$BIN" "$dir/pi-agent-lab" 2>/dev/null; then
		echo "linked $dir/pi-agent-lab -> $BIN"
		return 0
	fi
	return 1
}

append_once() {
	local file="$1" line="$2"
	mkdir -p "$(dirname "$file")"
	touch "$file"
	if grep -qF "$line" "$file"; then
		return 0
	fi
	printf '\n# pi-agent-lab on PATH\n%s\n' "$line" >>"$file"
	echo "added to $file:"
	echo "    $line"
}

install_path() {
	local existing
	if existing="$(command -v pi-agent-lab 2>/dev/null)" && [[ "$existing" -ef "$BIN" ]]; then
		return 0
	fi

	local -a system_dirs=()
	case "$(uname -s)" in
	Darwin)
		[[ "$(uname -m)" == arm64 ]] && system_dirs+=("/opt/homebrew/bin")
		system_dirs+=("/usr/local/bin")
		;;
	Linux)
		system_dirs+=("/usr/local/bin")
		;;
	*)
		echo "unrecognized OS ($(uname -s)); falling back to a user-local directory" >&2
		;;
	esac

	local dir
	for dir in "${system_dirs[@]}"; do
		if on_path "$dir" && writable_dir "$dir" && link_into "$dir"; then
			return 0
		fi
	done

	local user_dir="$HOME/.local/bin"
	mkdir -p "$user_dir"
	link_into "$user_dir" || return 1

	if on_path "$user_dir"; then
		echo "run 'hash -r' (bash/zsh) or open a new shell to pick it up"
		return 0
	fi

	local line='export PATH="$HOME/.local/bin:$PATH"'
	case "$(basename "${SHELL:-}")" in
	bash) append_once "$HOME/.bashrc" "$line" ;;
	zsh) append_once "$HOME/.zshrc" "$line" ;;
	fish) append_once "$HOME/.config/fish/config.fish" 'fish_add_path $HOME/.local/bin' ;;
	*) append_once "$HOME/.profile" "$line" ;;
	esac

	echo "open a new shell (or re-source your shell's startup file) for PATH to take effect"
}

install_path || echo "warning: adding pi-agent-lab to PATH failed; add $(dirname "$BIN") to your PATH manually" >&2

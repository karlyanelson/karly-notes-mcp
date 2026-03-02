#!/bin/bash
DIR="$(cd "$(dirname "$0")/.." && pwd)"
"$DIR/cli/karly-notes-cli" "$@"

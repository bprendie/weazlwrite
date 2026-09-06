#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

failed=0
while IFS= read -r file; do
  count=$(awk 'END { print NR }' "$file")
  if (( count > 300 )); then
    printf '%s: %s lines (ceiling: 300)\n' "$file" "$count"
    failed=1
  fi
done < <(rg --files -g '*.go' -g '*.sh' -g '*.ps1' -g '*.py' -g '*.js' -g '*.jsx' -g '*.ts' -g '*.tsx')

if (( failed == 0 )); then
  printf 'Source line ceiling passed (300 lines).\n'
fi
exit "$failed"

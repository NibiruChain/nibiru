#!/usr/bin/env bash
# Exercise the Wasm VM event policy without building libraries or using GitHub.
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
policy="$script_dir/wasmvm-ci-policy.sh"
count=0
while IFS='|' read -r name event ref changed tests package publish; do
  actual="$(bash "$policy" "$event" "$ref" "$changed")"
  expected="$(printf 'do-run=%s\ndo-package=%s\ndo-release=%s' "$tests" "$package" "$publish")"
  if [[ "$actual" != "$expected"$'\nreason='?* ]]; then
    printf 'FAIL: %s\nExpected:\n%s\nActual:\n%s\n' "$name" "$expected" "$actual" >&2
    exit 1
  fi
  count=$((count + 1))
done <<'CASES'
related PR|pull_request|refs/pull/1/merge|true|true|false|false
unrelated PR|pull_request|refs/pull/1/merge|false|false|false|false
related main push|push|refs/heads/main|true|true|false|false
unrelated main push|push|refs/heads/main|false|false|false|false
scheduled unchanged main|schedule|refs/heads/main|false|true|true|false
scheduled changed main|schedule|refs/heads/main|true|true|true|false
manual unchanged main|workflow_dispatch|refs/heads/main|false|true|true|false
manual changed main|workflow_dispatch|refs/heads/main|true|true|true|false
manual feature branch|workflow_dispatch|refs/heads/feature|false|true|true|false
release without path changes|push|refs/tags/lib/wasmvm/v1.13.2|false|true|true|true
release with path changes|push|refs/tags/lib/wasmvm/v1.13.2|true|true|true|true
optimizer tag|push|refs/tags/sai-wasm-optimizer/v0.1.0|false|false|false|false
chain tag|push|refs/tags/v2.21.2|true|false|false|false
malformed VM tag|push|refs/tags/lib/wasmvm/v1.2|false|false|false|false
candidate VM tag|push|refs/tags/lib/wasmvm/v1.2.3-rc.1|false|false|false|false
VM branch name is not a tag|push|refs/heads/lib/wasmvm/v1.2.3|true|false|false|false
feature push|push|refs/heads/feature|true|false|false|false
unsupported event|pull_request_target|refs/heads/main|true|false|false|false
CASES

for invalid in '' invalid TRUE; do
  if bash "$policy" push refs/heads/main "$invalid" >/dev/null 2>&1; then
    echo "FAIL: invalid change flag accepted: $invalid" >&2
    exit 1
  fi
done
if bash "$policy" push refs/heads/main >/dev/null 2>&1; then
  echo "FAIL: missing argument accepted" >&2
  exit 1
fi
if bash "$policy" push refs/heads/main false extra >/dev/null 2>&1; then
  echo "FAIL: extra argument accepted" >&2
  exit 1
fi
printf 'Passed %s policy cases and 5 invalid-input cases.\n' "$count"

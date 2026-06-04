#!/bin/sh
# sync.sh
# 
# 2 Modes:
# (Import) Grabs files from the local docs-nibiru repository.
# (Export) Exports files from the current repo to the docs-nibiru repository.

sync_pull() {
  local target="$1"
  echo "target: $target"
  local ext_path="../../../docs-nibiru/docs"
  cp -r $ext_path/$target $(dirname "$target")
}

main() {
  # echo "pwd: $(pwd)"
  echo "1: $1" 
  sync_pull $1
}

main "$1"

for dir in $(ls -d *); do
  echo "dir: $dir"
  main $dir
done
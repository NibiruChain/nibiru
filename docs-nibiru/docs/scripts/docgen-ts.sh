#!/usr/bin/env bash

set -eo pipefail

# NEW_DOCS_DIR is the ouput directory for the generated docs from nibijs.
NEW_DOCS_DIR="dev/tools/nibijs"
REPO_DIR=$(pwd)
if [ "$(basename "$PWD")" != "docs-nibiru" ]; then
  echo "pwd: $(pwd)"
  echo "Please run the script from docs-nibiru."
  exit 1
fi

# Fetch the nibijs markdown docs from the ts-sdk repo
clone_ts_sdk() {
  # Initialize the variable that will hold the flag status.
  local clone_flag=false

  # Loop over all command-line arguments.
  for i in $(seq 1 10); do
    local arg_var="yarn_exec_arg$i"
    arg=${!arg_var}
    if [ "$arg" == "--clone" ]; then
      # If the argument is "--clone", set the flag to true.
      clone_flag=true
      break
    fi
  done

  # Clone the repo if the flag was set.
  if $clone_flag; then
    rm -rf ts-sdk
    npx degit "NibiruChain/ts-sdk" ts-sdk --force
    # git clone git@github.com:NibiruChain/ts-sdk.git
  else
    echo "clone of ts-sdk is assumed to be present."
  fi
}

clone_ts_sdk

# Move the docs to NEW_DOCS_DIR
mkdir --parents $NEW_DOCS_DIR
cp -r ts-sdk/packages/nibijs/docs/** $NEW_DOCS_DIR

# From inside the NEW_DOCS_DIR, create README.md and nibijs.md
# README.md becomes the markdown for docs.nibiru.fi/dev/nibijs
# nibijs.md holds the content for both exports.md and functions.md
cp ts-sdk/packages/nibijs/README.md $NEW_DOCS_DIR/README.md
cd $NEW_DOCS_DIR
rm intro.md
egrep -lRZ 'intro.md' . | xargs -0 -l sed -i -e 's/intro.md/README.md/g'
mv modules.md nibijs.md
egrep -lRZ 'modules.md' . | xargs -0 -l sed -i -e 's/modules.md/nibijs.md/g'

# The moveFunction.ts script creates functions.md and exports.md using nibijs.md
cd "$REPO_DIR" # path back to the root
if [ "$(basename "$PWD")" == "$(basename "$REPO_DIR")" ]; then
  move_function_script="docs/scripts/moveFunction.ts"
  yarn run ts-node $move_function_script || npx ts-node $move_function_script
else
  echo "Current directory should be '$REPO_DIR'"
  echo "pwd: $(pwd)"
  exit 1
fi


egrep -lRZ 'nibijs.md' $NEW_DOCS_DIR/functions.md | xargs -0 -l sed -i -e 's/nibijs.md/functions.md/g'
egrep -lRZ 'nibijs.md' $NEW_DOCS_DIR/exports.md | xargs -0 -l sed -i -e 's/nibijs.md/exports.md/g'
egrep -lRZ 'nibijs.md' $NEW_DOCS_DIR/classes | xargs -0 -l sed -i -e 's/nibijs.md/README.md/g'
egrep -lRZ 'nibijs.md' $NEW_DOCS_DIR/enums | xargs -0 -l sed -i -e 's/nibijs.md/README.md/g'
egrep -lRZ 'nibijs.md' $NEW_DOCS_DIR/interfaces | xargs -0 -l sed -i -e 's/nibijs.md/README.md/g'

rm -rf docs/$NEW_DOCS_DIR
mv $NEW_DOCS_DIR docs/$NEW_DOCS_DIR

# Cleanup
rm -rf ts-sdk


# Q: How does the egrep xargs command work?
#
# ```bash
# egrep -lRZ "\.jpg|\.png|\.gif" . \
#   | xargs -0 -l sed -i -e 's/\.jpg\|\.gif\|\.png/.bmp/g'
# ```
# The above example finds all instances of ".jpg", ".png", and ".gif" in any files at
# the current wording directory recursively and replaces them with ".bmp"
#
# ### **`egrep`**: find matching lines using extended regular expressions
# * `-l`: only list matching filenames
# * `-R`: search recursively through all given directories
# * `-Z`: use `\0` as record separator
# * `"\.jpg|\.png|\.gif"`: match one of the strings `".jpg"`, `".gif"` or `".png"`
# * `.`: start the search in the current directory
#
#
# ### **`xargs`**: execute a command with the stdin as argument
# * `-0`: use `\0` as record separator. This is important to match the `-Z` of `egrep` and to avoid being fooled by spaces and newlines in input filenames.
# * `-l`: use one line per command as parameter
#
# ### **`sed`**: the **s**tream **ed**itor
# * `-i`: replace the input file with the output without making a backup
# * `-e`: use the following argument as expression
# * `'s/\.jpg\|\.gif\|\.png/.bmp/g'`: replace all occurrences of the strings `".jpg"`, `".gif"` or `".png"` with `".bmp"`

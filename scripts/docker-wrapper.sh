#!/usr/bin/env sh

##
## Input parameters
##
BINARY=/nibid/${BINARY:-nibid}
ID=${ID:-0}
LOG=${LOG:-nibid.log}

##
## Assert linux binary
##
if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'nibid' E.g.: -e BINARY=nibid_my_test_version"
	exit 1
fi
BINARY_CHECK="$(file "$BINARY" | grep 'ELF 64-bit LSB executable, x86-64')"
if [ -z "${BINARY_CHECK}" ]; then
	echo "Binary needs to be OS linux, ARCH amd64"
	exit 1
fi

##
## Run binary with all parameters
##
export NIBIDHOME="/nibid/node${ID}/nibid"

if [ -d "$(dirname "${NIBIDHOME}"/"${LOG}")" ]; then
  "${BINARY}" --home "${NIBIDHOME}" "$@" | tee "${NIBIDHOME}/${LOG}"
else
  "${BINARY}" --home "${NIBIDHOME}" "$@"
fi


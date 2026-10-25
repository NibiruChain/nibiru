package version

const (

	// The default version of TMCoreSemVer is the value used as the
	// fallback version of CometBFT when not using git describe.
	// It is formatted with semantic versioning.
	TMCoreSemVer = "0.37.18"
	// ABCISemVer is the semantic version of the ABCI protocol
	ABCISemVer  = "1.0.0"
	ABCIVersion = ABCISemVer
	// P2PProtocol versions all p2p behavior and msgs.
	// This includes proposer selection.
	P2PProtocol uint64 = 8

	// BlockProtocol versions all block data structures and processing.
	// This includes validity of blocks and state updates.
	BlockProtocol uint64 = 11
)

// NibiruBFTSemVer is the version of Nibiru's CometBFT fork (Nibiru BFT). It
// tracks the nibid release version in lockstep and is set via ldflags at
// build time (contrib/scripts/build-nibiru.sh). The default is the first
// nibid release that ships the fork.
var NibiruBFTSemVer = "2.22.0"

// TMGitCommitHash is the git commit of the build, logged as "commit_hash" at
// node startup. It is set via ldflags (contrib/scripts/build-nibiru.sh for
// nibid, the bft justfile for the test-only cometbft binary).
var TMGitCommitHash = ""

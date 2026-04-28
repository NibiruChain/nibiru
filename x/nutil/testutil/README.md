# Testutil Directory

The x/nutil/testutil directory is not a Cosmos SDK module itself. It is a
collection of test utilities for testing the other x/modules.

For app-backed tests, prefer the helpers in `testapp` for app and context setup.
Use table-driven tests or `testify` suites for structure, and use the block
helpers in `testapp/block.go` when tests need to advance block height or time.
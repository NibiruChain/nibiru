UNAME_OS              := $(shell uname -s)
UNAME_ARCH            := $(shell uname -m)

include contrib/make/build.mk
include contrib/make/chaosnet.mk
include contrib/make/localnet.mk
include contrib/make/proto.mk
include contrib/make/mock.mk
include contrib/make/lint.mk
include contrib/make/test.mk
include contrib/make/simulation.mk
include contrib/make/release.mk


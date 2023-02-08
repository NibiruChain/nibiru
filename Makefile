UNAME_OS              := $(shell uname -s)
UNAME_ARCH            := $(shell uname -m)

include contrib/make/*.mk

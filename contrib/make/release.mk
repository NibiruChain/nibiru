###############################################################################
###                               Release                                   ###
###############################################################################

release:
	docker run --rm -v "$(CURDIR)":/code -w /code goreleaser/goreleaser-cross --skip-publish --rm-dist
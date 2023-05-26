###############################################################################
###                            Lint                                         ###
###############################################################################

.PHONY: lint
lint:
	docker run -v $(CURDIR):/code --rm -w /code golangci/golangci-lint:v1.52.2-alpine golangci-lint run

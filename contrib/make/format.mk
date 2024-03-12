format:
	@echo "--> Formating code and ordering imports"
	@goimports -local github.com/NibiruChain -w .
	@gofmt -w .
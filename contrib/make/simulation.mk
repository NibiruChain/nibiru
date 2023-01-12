BINDIR = $(GOPATH)/bin
RUNSIM = $(BINDIR)/runsim
SIMAPP = ./simapp

.PHONY: runsim
runsim: $(RUNSIM)
$(RUNSIM):
	@echo "Installing runsim..."
	@(cd /tmp && go install github.com/cosmos/tools/cmd/runsim@v1.0.0)

.PHONY: test-sim-nondeterminism
test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=100 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h

.PHONY: test-sim-default-genesis-fast
test-sim-default-genesis-fast:
	@echo "Running default genesis simulation..."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation  \
		-Enabled=true -NumBlocks=100 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v

.PHONY: test-sim-custom-genesis-multi-seed
test-sim-custom-genesis-multi-seed: runsim
	@echo "Running multi-seed custom genesis simulation..."
	@$(RUNSIM) -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation

.PHONY: test-sim-multi-seed-long
test-sim-multi-seed-long: runsim
	@echo "Running long multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation

.PHONY: test-sim-multi-seed-short
test-sim-multi-seed-short: runsim
	@echo "Running short multi-seed application simulation. This may take awhile!"
	@$(RUNSIM) -Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation

.PHONY: test-sim-benchmark-invariants
test-sim-benchmark-invariants:
	@echo "Running simulation invariant benchmarks..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkInvariants -run=^$ \
	-Enabled=true -NumBlocks=1000 -BlockSize=200 \
	-Period=1 -Commit=true -Seed=57 -v -timeout 24h
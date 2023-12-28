SIMAPP = ./simapp

.PHONY: test-sim-nondeterminism
test-sim-nondeterminism:
	@echo "Running non-determinism test..."
	@go test -mod=readonly -v $(SIMAPP) \
		-run TestAppStateDeterminism \
		-Enabled=true \
		-Params=params.json \
		-NumBlocks=100 \
		-BlockSize=200 \
		-Commit=true \
		-Period=0 \
		-Verbose=true

.PHONY: test-sim-default-genesis-fast
test-sim-default-genesis-fast:
	@echo "Running default genesis simulation..."
	@go test -mod=readonly -v $(SIMAPP) \
		-run TestFullAppSimulation \
		-Params=params.json \
		-Enabled=true \
		-NumBlocks=100 \
		-BlockSize=200 \
		-Commit=true \
		-Seed=99 \
		-Period=0

.PHONY: test-sim-import-export
test-sim-import-export:
	@echo "Running application import/export simulation. This may take several minutes..."
	@go test -mod=readonly -v $(SIMAPP) \
		-run TestAppImportExport \
		-Params=params.json \
		-Enabled=true \
		-NumBlocks=100 \
		-Commit=true \
		-Seed=99 \
		-Period=5

.PHONY: test-sim-after-import
test-sim-after-import:
	@echo "Running application simulation-after-import. This may take several minutes..."
	@go test -mod=readonly -v $(SIMAPP) \
		-run TestAppSimulationAfterImport \
		-Params=params.json \
		-Enabled=true \
		-NumBlocks=50 \
		-Commit=true \
		-Seed=99 \
		-Period=5

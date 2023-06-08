###############################################################################
###                                  Proto                                  ###
###############################################################################

containerProtoVer=v0.3
containerProtoImage=tendermintdev/sdk-proto-gen:$(containerProtoVer)
containerProtoGen=cosmos-sdk-proto-gen-$(containerProtoVer)
containerProtoGenSwagger=cosmos-sdk-proto-gen-swagger-$(containerProtoVer)
containerProtoFmt=cosmos-sdk-proto-fmt-$(containerProtoVer)

.PHONY: proto-gen
proto-gen:
	@echo "Generating Protobuf files"
	@if docker ps -a --format '{{.Names}}' | grep -Eq "^${containerProtoGen}$$"; then \
		docker start -a $(containerProtoGen); else \
		docker run --name $(containerProtoGen) \
			-v "$(CURDIR)":/workspace \
			-w /workspace \
			$(containerProtoImage) \
			sh ./contrib/scripts/protocgen.sh; \
	fi

# How to run manually:
# docker build --pull --rm -f "contrib/devtools/Dockerfile" -t cosmossdk-proto:latest "contrib/devtools"
# docker run --rm -v $(pwd):/workspace --workdir /workspace cosmossdk-proto sh ./scripts/protocgen.sh

proto-gen2:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh


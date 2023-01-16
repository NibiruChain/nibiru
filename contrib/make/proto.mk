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

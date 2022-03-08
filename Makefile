.PHONY: proto

proto:
	docker build -t dev:proto --progress="plain" -f ./contrib/proto.dockerfile .
	docker run -v "$(CURDIR):/work" -w /work/proto dev:proto buf mod update
	docker run -v "$(CURDIR):/work" -w /work/proto dev:proto buf generate --template buf.gen.yaml
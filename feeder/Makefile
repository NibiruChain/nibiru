docker:
	cd ..; GOOS="linux" GOARCH="arm64" make build
	GOOS="linux" GOARCH="arm64" go build -o build/feeder ./cmd/feeder/.
	cd ..; docker build --progress plain -t feeder:master -f ./feeder/Dockerfile .
	rm -rf ../build/nibid
	rm -rf ./build/feeder

docker-compose: docker
	docker-compose down --volumes --rmi all
	docker-compose up
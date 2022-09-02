FROM golang:1.19-alpine3.16 AS build

WORKDIR /build

COPY ./ ./

WORKDIR /build/feeder
RUN CGO_ENABLED=0 go build -o feeder ./cmd/feeder/.

FROM alpine:3.16

WORKDIR /app

COPY --from=build /build/feeder/feeder ./feeder

RUN chmod +x feeder

CMD ["./feeder"]
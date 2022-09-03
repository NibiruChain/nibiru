FROM alpine:3.16.2
WORKDIR /root/

COPY ./build/nibid /usr/local/bin/nibid
COPY ./feeder/build/feeder /usr/local/bin/feeder

COPY ./feeder/scripts/. /app/.
RUN chmod +x /app/node.sh
RUN chmod +x /app/oracle.sh

WORKDIR /app
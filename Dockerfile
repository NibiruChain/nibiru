FROM ubuntu:18.04

RUN apt-get update && \
    apt-get -y upgrade && \
    apt-get -y install curl jq file

VOLUME [ /nibid ]
WORKDIR /nibid
EXPOSE 26656 26657
ENTRYPOINT ["/usr/bin/docker-wrapper.sh"]
CMD ["start"]
STOPSIGNAL SIGTERM

COPY scripts/docker-wrapper.sh /usr/bin/docker-wrapper.sh


FROM alpine
WORKDIR /root
RUN apk --no-cache add ca-certificates
COPY nibid /usr/bin/nibid
ENTRYPOINT ["/usr/bin/nibid"]
CMD [ "start" ]
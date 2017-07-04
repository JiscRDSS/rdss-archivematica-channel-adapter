FROM golang:1.8.3-alpine as builder
WORKDIR /go/src/github.com/JiscRDSS/rdss-archivematica-channel-adapter
COPY . .
RUN set -x \
	&& apk add --no-cache --virtual .build-deps make gcc musl-dev \
	&& make test vet \
	&& make build

FROM alpine:3.6
WORKDIR /var/lib/archivematica
COPY --from=builder /go/src/github.com/JiscRDSS/rdss-archivematica-channel-adapter/rdss-archivematica-channel-adapter .
RUN apk --no-cache add ca-certificates
RUN addgroup -g 333 -S archivematica && adduser -u 333 -h /var/lib/archivematica -S -G archivematica archivematica
USER archivematica
ENTRYPOINT ["/var/lib/archivematica/rdss-archivematica-channel-adapter"]
CMD ["help"]

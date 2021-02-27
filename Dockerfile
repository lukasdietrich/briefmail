FROM golang:alpine as build
	WORKDIR /build
	COPY . .

	RUN apk --no-cache add build-base git \
		&& go get -v github.com/google/wire/cmd/wire \
		&& make build

FROM alpine:latest
	WORKDIR /app
	COPY --from=build /build/target/briefmail ./
	COPY LICENSE README.md ./

	RUN apk --no-cache add ca-certificates

	VOLUME [ "/data" ]

	ENV BRIEFMAIL_LOG_LEVEL=DEBUG \
		BRIEFMAIL_STORAGE_BLOBS_FOLDERNAME=/data/blobs \
		BRIEFMAIL_STORAGE_CACHE_FOLDERNAME=/data/cache \
		BRIEFMAIL_STORAGE_DATABASE_FILENAME=/data/briefmail.sqlite

	EXPOSE 25/tcp 587/tcp 110/tcp 995/tcp

	ENTRYPOINT [ "/app/briefmail", "--config", "/data/config.toml" ]
	CMD [ "start" ]

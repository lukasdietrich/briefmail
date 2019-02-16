FROM golang:alpine as build

WORKDIR /temp/build
COPY . .

RUN apk --no-cache add build-base git \
	&& make clean build

FROM alpine:latest

WORKDIR /app
COPY --from=build /temp/build/target/* /app/
COPY _example/* /config/

RUN apk --no-cache add ca-certificates

VOLUME [ \
	"/data",  \
	"/config" \
]

EXPOSE 25/tcp
EXPOSE 587/tcp
EXPOSE 110/tcp
EXPOSE 995/tcp

CMD [ \
	"./briefmail",                              \
	"--data", "/data",                          \
	"start",                                    \
	"--config", "/config/config.toml",          \
	"--addressbook", "/config/addressbook.toml" \
]

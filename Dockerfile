FROM golang:alpine as build

WORKDIR /temp/build
COPY . .

RUN apk --no-cache add build-base git && make clean build

FROM alpine:latest

WORKDIR /opt/briefmail
COPY --from=build /temp/build/target/briefmail /opt/briefmail/bin/briefmail

RUN apk --no-cache add ca-certificates

VOLUME [ "/opt/briefmail/etc", "/opt/briefmail/var" ]

ENV BRIEFMAIL_CONFIG /opt/briefmail/etc/config.toml
ENV BRIEFMAIL_ADDRESSBOOK /opt/briefmail/etc/addressbook.toml
ENV BRIEFMAIL_DATA /opt/briefmail/var
ENV PATH /opt/briefmail/bin:${PATH}

EXPOSE 25/tcp
EXPOSE 587/tcp
EXPOSE 110/tcp
EXPOSE 995/tcp

CMD [ "briefmail", "start" ]

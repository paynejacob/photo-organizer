FROM golang:1.17 as build

WORKDIR /build

ADD pkg pkg
ADD go.* .
ADD main.go main.go

RUN go build -o photo-organizer main.go
RUN chmod +x photo-organizer

FROM ubuntu

RUN apt-get update && apt-get install -y exiftool

COPY --from=build /build/photo-organizer /usr/local/bin/

ENTRYPOINT ["photo-organizer"]
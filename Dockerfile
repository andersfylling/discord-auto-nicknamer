FROM golang:1.16.2-alpine3.13 as builder
MAINTAINER https://github.com/andersfylling
WORKDIR /build
COPY . /build
RUN go test ./...
RUN cd cmd/bot && go build -o discordbot
CMD ["cmd/bot/discordbot"]
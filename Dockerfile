FROM golang:1.20-alpine3.18 AS build-stage
RUN apk update && apk upgrade && apk add build-base

WORKDIR /camino-messenger-bot

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN apk add olm-dev

RUN go build -o bot  cmd/camino-messenger-bot/main.go


#runtime stage
FROM alpine:3.18 AS runtime-stage

RUN apk add --no-cache olm-dev

WORKDIR /

ARG CONFIG_CONTENT
RUN echo "$CONFIG_CONTENT" > camino-messenger-bot.yaml
COPY --from=build-stage /camino-messenger-bot/bot /camino-messenger-bot

#rpc server port
RUN rpc_server_port=$(awk '/rpc-server-port:/ {print $2}' camino-messenger-bot.yaml)
EXPOSE $rpc_server_port

CMD ["./camino-messenger-bot", "config", "camino-messenger-bot.yaml"]
FROM golang:1.23.5-alpine3.20 AS build-stage
RUN apk update && apk upgrade && apk add build-base

WORKDIR /camino-messenger-bot

# add ext library
RUN apk add olm-dev

# build
COPY . .
RUN apk --no-cache add git bash
RUN git submodule update --init
RUN CAMINOBOT_PATH=$(pwd) bash ./scripts/build.sh


#runtime stage
FROM alpine:3.20 AS runtime-stage

RUN apk add --no-cache olm-dev

WORKDIR /

COPY --from=build-stage /camino-messenger-bot/build/camino-messenger-bot /camino-messenger-bot
COPY --from=build-stage /camino-messenger-bot/migrations ./migrations

ENTRYPOINT ["./camino-messenger-bot"]

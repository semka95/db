# Builder
FROM golang:1.17.6-alpine3.15 as builder

RUN apk update && apk upgrade && \
    apk --update add git make

WORKDIR /app

COPY go.* ./
RUN go mod download

COPY . .

RUN make engine

# Distribution
FROM alpine:3.15.0

RUN apk update && apk upgrade && \
    apk --update --no-cache add tzdata && \
    mkdir /app 

WORKDIR /app 

EXPOSE 9000

COPY --from=builder /app/engine /app

CMD /app/engine

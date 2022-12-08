FROM golang:1.19-alpine3.16 as builder

ENV GOPROXY https://goproxy.cn
WORKDIR /mayfly
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source for building server
COPY . .

# Build
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux \
    go build -a \
    -o mayfly-go main.go

FROM alpine:3.16

RUN apk add --no-cache ca-certificates bash expat

WORKDIR /mayfly

COPY --from=builder /mayfly/config.yml /mayfly/config.yml
COPY --from=builder /mayfly/mayfly-go /usr/local/bin/mayfly-go

CMD ["mayfly-go"]

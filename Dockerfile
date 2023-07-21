# 构建前端资源
FROM node:18-alpine as fe-builder

WORKDIR /mayfly

COPY mayfly_go_web .

RUN yarn

RUN yarn build

# 构建后端资源
FROM golang:1.19-alpine3.16 as be-builder

ENV GOPROXY https://goproxy.cn
WORKDIR /mayfly

# Copy the go source for building server
COPY server .

RUN go mod download

COPY --from=fe-builder /mayfly/dist /mayfly/static/static

# Build
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux \
    go build -a \
    -o mayfly-go main.go

FROM alpine:3.16

RUN apk add --no-cache ca-certificates bash expat

WORKDIR /mayfly

COPY --from=be-builder /mayfly/mayfly-go /usr/local/bin/mayfly-go

CMD ["mayfly-go"]

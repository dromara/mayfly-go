ARG BASEIMAGES=m.daocloud.io/docker.io/alpine:3.20.2

FROM ${BASEIMAGES} AS builder
ARG TARGETARCH

ARG MAYFLY_GO_VERSION
ARG MAYFLY_GO_DIR_NAME=mayfly-go-linux-${TARGETARCH}
ARG MAYFLY_GO_URL=https://gitee.com/dromara/mayfly-go/releases/download/${MAYFLY_GO_VERSION}/${MAYFLY_GO_DIR_NAME}.zip

RUN wget -cO mayfly-go.zip ${MAYFLY_GO_URL} && \
    unzip mayfly-go.zip && \
    mv ${MAYFLY_GO_DIR_NAME}/* /opt


FROM ${BASEIMAGES}

ARG TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

COPY --from=builder /opt/mayfly-go /usr/local/bin/mayfly-go

WORKDIR /mayfly-go

EXPOSE 18888

CMD ["mayfly-go"]
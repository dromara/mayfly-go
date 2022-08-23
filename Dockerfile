FROM  debian:buster
COPY mayfly-go-linux-amd64/ /mayfly/
COPY sources.list /etc/apt/sources.list
RUN chmod +x /mayfly/mayfly-go && chmod +x /mayfly/startup.sh && apt update && apt install -y procps curl 
WORKDIR /mayfly
CMD ["/mayfly/mayfly-go"]

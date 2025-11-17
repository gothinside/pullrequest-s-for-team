FROM golang:1.24

WORKDIR ${GOPATH}/avito-pr/
COPY . ${GOPATH}/avito-pr/

RUN go build -o /build ./cmd \
    && go clean -cache -modcache

EXPOSE 8080

CMD ["/build"]
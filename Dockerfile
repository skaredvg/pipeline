FROM golang
RUN mkdir /go/src/pipeline
WORKDIR /go/src/pipeline
ADD main.go .
ADD go.mod .
RUN go build .

FROM alpine
LABEL version = '1.1'
LABEL maintainer = 'skaredvg72@gmail.com'
WORKDIR /root/
COPY --from=0 /go/src/pipeline .
ENTRYPOINT ./pipeline

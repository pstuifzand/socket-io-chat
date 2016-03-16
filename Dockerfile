FROM golang
WORKDIR /go/src/github.com/pstuifzand/socket-io-chat
ADD . /go/src/github.com/pstuifzand/socket-io-chat
RUN go get
RUN go build
EXPOSE 5000

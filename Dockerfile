FROM golang
WORKDIR /go/src/stuifzand.eu/chat
ADD . /go/src/stuifzand.eu/chat
RUN go get
RUN go build

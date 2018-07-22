FROM golang:1.8

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 80

CMD ["app", "-blacklist_ranges", "10.0.0.0/8 172.16.0.0/12 192.168.0.0/16", "-host", "0.0.0.0", "-port", "80"]

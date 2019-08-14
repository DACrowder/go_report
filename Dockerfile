FROM golang:1.12.7 as builder

LABEL maintainer="Alex Crowder <alex.crowder@ucalgary.ca"

WORKDIR $GOPATH/src/go_report

COPY . .

RUN go get -d -v ./...

RUN go install -v ./...

RUN go get -u github.com/kardianos/govendor

RUN govendor install +vendor,^program

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go/bin/go-report .


###########

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /go/bin/go-report .

EXPOSE 8080

CMD ["go-report"]

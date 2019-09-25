FROM golang:1.12.7 as builder
RUN go get -u github.com/kardianos/govendor
WORKDIR $GOPATH/src/go_report/
COPY . .
RUN govendor sync
RUN govendor install +vendor,^program
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo  -o /go_report .

########### 

FROM alpine:latest
WORKDIR /
RUN addgroup -S reporters && adduser -S goreporter -G reporters
USER goreporter
COPY --from=builder /go_report /home/goreporter/go_report
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
EXPOSE 8080
ENTRYPOINT ["/home/goreporter/go_report"]

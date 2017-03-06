FROM golang:1.7
COPY ./cmd/negotiator/negotiator /go/bin/negotiator

CMD ["negotiator"]

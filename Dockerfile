FROM golang:1.7-alpine
COPY ./cmd/negotiator/negotiator /go/bin/negotiator

CMD ["negotiator"]

FROM golang:1.7-alpine
COPY ./negotiator /go/bin/negotiator

CMD ["negotiator"]

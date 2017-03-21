FROM golang:1.7
COPY ./cmd/negotiator/negotiator /go/bin/negotiator
COPY ./cmd/jobs/jobs /go/bin/jobs

CMD ["negotiator"]

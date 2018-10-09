
FROM golang:1.11-alpine

ENTRYPOINT ["/bin/git-contrib-graph"]

RUN apk add --no-cache git

COPY git-contrib-graph.go /app/
COPY go.mod /app/
COPY go.sum /app/

WORKDIR /app/

ENV GO111MODULE=on CGO_ENABLED=0
RUN go build -o /bin/git-contrib-graph /app/git-contrib-graph.go

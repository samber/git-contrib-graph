
FROM golang:1.11-alpine

ENTRYPOINT ["/bin/git-contrib-graph"]

RUN apk add --no-cache git

COPY git-contrib-graph.go /app/

RUN go get gopkg.in/src-d/go-git.v4 \
    && go build -o /bin/git-contrib-graph /app/git-contrib-graph.go

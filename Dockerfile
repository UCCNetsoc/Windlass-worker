FROM golang:1.13 AS dev

WORKDIR /windlass-worker

RUN go get github.com/go-task/task/cmd/task \
    github.com/derekparker/delve/cmd/dlv \
    github.com/nomad-software/vend

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . . 

RUN go install github.com/UCCNetworkingSociety/Windlass-worker/cmd/windlass-worker

RUN go mod vendor && vend

CMD [ "go", "run", "cmd/windlass-worker/main.go" ]

FROM alpine

WORKDIR /bin

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

COPY --from=dev /go/bin/windlass-worker ./windlass-worker

CMD [ "windlass-worker" ]
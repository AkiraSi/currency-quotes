ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY common ./common
COPY configs/worker ./configs/worker
COPY infrastructure/worker ./infrastructure/worker
COPY pkg ./pkg

RUN go build -o worker-app ./infrastructure/worker/

CMD ["./worker-app"]

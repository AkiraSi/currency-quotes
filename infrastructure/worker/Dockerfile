ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine

WORKDIR /app

COPY go.mod go.sum ./

COPY ../../../currency-quotes/infrastructure/worker .

RUN go build -o worker-app ./infrastructure/worker/

CMD ["./worker-app"]

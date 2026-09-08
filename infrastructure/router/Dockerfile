ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY common ./common
COPY configs/router ./configs/router
COPY infrastructure/router ./infrastructure/router

RUN go build -o router-app ./infrastructure/router/

EXPOSE 3000

CMD ["./router-app"]

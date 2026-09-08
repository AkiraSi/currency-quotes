ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine

WORKDIR /app

COPY go.mod go.sum ./

COPY ../../../currency-quotes/infrastructure/router .

RUN go build -o router-app ./infrastructure/router/

EXPOSE 3000-3000

CMD ["./router-app"]

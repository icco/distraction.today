FROM golang:1.24-alpine

ENV PORT=8080
EXPOSE 8080

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY *.go .
COPY static static
COPY templates templates

RUN go build -v -o /usr/local/bin/server .

CMD ["server"]

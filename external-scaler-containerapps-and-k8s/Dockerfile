FROM golang:1.23-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o external-scaler .


FROM alpine:latest

WORKDIR /

COPY --from=builder /src/external-scaler .

ENTRYPOINT ["/external-scaler"]
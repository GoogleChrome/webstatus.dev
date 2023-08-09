FROM golang:1.20-alpine3.18 as builder

WORKDIR /work
COPY . .

RUN go mod download && go build

FROM alpine:3.18

COPY --from=builder /work/sample .

CMD ./sample
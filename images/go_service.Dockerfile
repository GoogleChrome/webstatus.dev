FROM golang:1.21-alpine3.18 as builder

WORKDIR /work
COPY lib/go.mod lib/go.sum lib/
ARG service_dir
COPY ${service_dir}/go.mod ${service_dir}/go.sum ${service_dir}/
RUN go work init && \
    go work use ./lib && \
    go work use ${service_dir} ${service_dir}
WORKDIR /work/${service_dir}
RUN  go mod download
WORKDIR /work
COPY lib lib
COPY ${service_dir} ${service_dir}
RUN go build -o server ./${service_dir}/cmd/server

FROM alpine:3.18

COPY --from=builder /work/server .

CMD ./server
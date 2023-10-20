# Copyright 2023 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
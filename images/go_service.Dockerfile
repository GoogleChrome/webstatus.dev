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

FROM golang:1.24.0-alpine3.21 AS builder

WORKDIR /work

# Cache the layers for the dependencies in the lib module.
# These layers are common among most Go services so each can re-use them.
COPY lib/go.mod lib/go.sum lib/
COPY lib/gen/go.mod lib/gen/go.sum lib/gen/
RUN go work init && \
    go work use ./lib && \
    go work use ./lib/gen
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Create the layers for the specific service in ${service_dir}.
ARG service_dir
COPY ${service_dir}/go.mod ${service_dir}/go.sum ${service_dir}/
RUN  go work use ${service_dir} ${service_dir}
WORKDIR /work/${service_dir}
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the source files now that all the dependencies have been installed.
WORKDIR /work
COPY lib lib
COPY ${service_dir} ${service_dir}

# Build the binary
ARG SKAFFOLD_GO_GCFLAGS
ARG MAIN_BINARY=server
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o program ./${service_dir}/cmd/${MAIN_BINARY}

FROM alpine:3.21

# Copy only the binary from the previous image
COPY --from=builder /work/program .

# Assuming that service has a binary called server, make that the command to run when the image starts.
CMD ["./program"]
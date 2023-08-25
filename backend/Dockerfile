# Copyright 2023 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.20.7-bullseye as builder

WORKDIR /app

COPY go.* ./

RUN go mod download

COPY . ./

RUN go build -o server ./cmd/server

FROM debian:bullseye-slim as release

COPY --from=builder /app/server /server
CMD ["/server"]

FROM release as docker_local_release
ENV SPANNER_DATABASE=
ENV SPANNER_EMULATOR_HOST=spanner:9010
ENV PROJECT_ID=local
ENV FIRESTORE_DATABASE=
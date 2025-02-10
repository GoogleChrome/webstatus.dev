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

FROM google/cloud-sdk:509.0.0-emulators
# TARGETARCH is an automatic platform ARG enabled by Docker BuildKit.
# https://docs.docker.com/reference/dockerfile/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH

ENV SPANNER_PROJECT_ID=local
ENV SPANNER_INSTANCE_ID=local
ENV SPANNER_DATABASE_ID=local
ENV SPANNER_EMULATOR_HOST=0.0.0.0:9010

# Install Wrench - https://github.com/cloudspannerecosystem/wrench
ENV WRENCH_VERSION="1.10.1"
RUN echo "Downloading for version ${WRENCH_VERSION} for architecture ${TARGETARCH}"
RUN curl \
    -L -o wrench.tar.gz \
    "https://github.com/cloudspannerecosystem/wrench/releases/download/v${WRENCH_VERSION}/wrench-${WRENCH_VERSION}-linux-${TARGETARCH}.tar.gz" && \
    tar -xf wrench.tar.gz && mv wrench /bin/

RUN gcloud config set auth/disable_credentials true && \
    gcloud config set project ${SPANNER_PROJECT_ID} && \
    gcloud config set api_endpoint_overrides/spanner http://localhost:9020/ && \
    mkdir schemas

COPY ./.dev/spanner/run.sh run.sh
COPY ./infra/storage/spanner/ schemas/

ENTRYPOINT [ "./run.sh" ]

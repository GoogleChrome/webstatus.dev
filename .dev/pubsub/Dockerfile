# Copyright 2025 Google LLC
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

FROM google/cloud-sdk:551.0.0-emulators
# TARGETARCH is an automatic platform ARG enabled by Docker BuildKit.
# https://docs.docker.com/reference/dockerfile/#automatic-platform-args-in-the-global-scope
ARG TARGETARCH

ENV PROJECT_ID=local
ENV PORT=8060

COPY ./.dev/pubsub/run.sh run.sh

ENTRYPOINT [ "./run.sh" ]

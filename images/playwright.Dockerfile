# Copyright 2026 Google LLC
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

FROM mcr.microsoft.com/devcontainers/base:trixie

# Install Node.js 24.11.1
RUN apt-get update && apt-get install -y wget xz-utils && \
    wget https://nodejs.org/dist/v24.11.1/node-v24.11.1-linux-x64.tar.xz && \
    tar -xJf node-v24.11.1-linux-x64.tar.xz -C /usr/local --strip-components=1 && \
    rm node-v24.11.1-linux-x64.tar.xz

WORKDIR /work

EXPOSE 4444

# We don't specify a CMD here because we will pass the command at runtime
# to use the correct Playwright version dynamically.

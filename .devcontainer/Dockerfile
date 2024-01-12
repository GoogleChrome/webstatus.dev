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

# The Node version from https://mcr.microsoft.com/en-us/product/devcontainers/typescript-node/tags
ARG VARIANT="20"
FROM mcr.microsoft.com/devcontainers/typescript-node:1-${VARIANT}

ENV CLOUD_SDK_VERSION=459.0.0
# Install gcloud similarly to how it is done in cloud-sdk-docker
# https://github.com/GoogleCloudPlatform/cloud-sdk-docker/blob/master/debian_component_based/Dockerfile
RUN if [ `uname -m` = 'x86_64' ]; then echo -n "x86_64" > /tmp/arch; else echo -n "arm" > /tmp/arch; fi;
RUN ARCH=`cat /tmp/arch` && curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${CLOUD_SDK_VERSION}-linux-${ARCH}.tar.gz && \
    tar xzf google-cloud-cli-${CLOUD_SDK_VERSION}-linux-${ARCH}.tar.gz && \
    rm google-cloud-cli-${CLOUD_SDK_VERSION}-linux-${ARCH}.tar.gz
RUN echo -n "alpha beta gke-gcloud-auth-plugin cloud-run-proxy" > /tmp/additional_components
RUN /google-cloud-sdk/install.sh --usage-reporting=false \
    --additional-components `cat /tmp/additional_components` && rm -rf /google-cloud-sdk/.install/.backup

# Switch to the default user and append to PATH
USER node

RUN echo 'PATH=${PATH}:/google-cloud-sdk/bin' >> ~/.bashrc

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

FROM node:22.3.0-alpine3.20 as base

FROM base as builder

WORKDIR /work
ARG service_dir
ARG build_env
ENV BUILD_ENV ${build_env}
COPY package.json package.json
COPY package-lock.json package-lock.json
COPY ${service_dir}/package.json ${service_dir}/package.json
COPY lib/gen/openapi/ lib/gen/openapi/
WORKDIR /work/${service_dir}
RUN --mount=type=cache,target=/root/.npm \
    npm install --ignore-scripts --include-workspace-root=true
COPY ${service_dir}/ /work/${service_dir}/
RUN npm run postinstall || true
RUN npm run build


FROM base as production

WORKDIR /work
ARG service_dir
COPY --from=builder /work/package.json /work/package.json
COPY --from=builder /work/package-lock.json /work/package-lock.json
COPY --from=builder /work/${service_dir}/package.json /work/${service_dir}/package.json
WORKDIR /work/${service_dir}
RUN --mount=type=cache,target=/root/.npm \
    npm install --ignore-scripts --production
RUN ln -s /work/node_modules /work/${service_dir}/node_modules
COPY --from=builder /work/${service_dir}/dist /work/${service_dir}/dist
CMD npm run start

FROM nginx:alpine3.19-slim as placeholder

ARG service_dir
COPY --from=builder /work/${service_dir}/nginx.conf /etc/nginx/nginx.conf
COPY --from=builder /work/${service_dir}/placeholder/static /usr/share/nginx/html
COPY --from=builder /work/${service_dir}/scripts/setup_server.sh /docker-entrypoint.d/setup_server.sh

FROM nginx:alpine3.19-slim as static

ARG service_dir
COPY --from=builder /work/${service_dir}/nginx.conf /etc/nginx/nginx.conf
COPY --from=production /work/${service_dir}/dist/static /usr/share/nginx/html
COPY --from=builder /work/${service_dir}/scripts/setup_server.sh /docker-entrypoint.d/setup_server.sh

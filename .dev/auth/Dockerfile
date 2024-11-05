# Copyright 2024 Google LLC
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

FROM node:22.11.0-slim
# FIREBASE_CLI_VERSION from https://github.com/firebase/firebase-tools/releases
ARG FIREBASE_CLI_VERSION=13.20.1
# Workaround for https://github.com/firebase/firebase-tools/issues/6931 is to install via npm
RUN npm i firebase-tools@${FIREBASE_CLI_VERSION} -g
# Once https://github.com/firebase/firebase-tools/issues/6931 is resolved, we can install via ADD instead of npm
# ADD https://github.com/firebase/firebase-tools/releases/download/v${FIREBASE_CLI_VERSION}/firebase-tools-linux /usr/local/bin/firebase
RUN chmod +x /usr/local/bin/firebase
ADD firebase.json firebase.json
RUN firebase setup:emulators:ui --project=local
ENTRYPOINT [ "firebase", "emulators:start", "--project=local" ]
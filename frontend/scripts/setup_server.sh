#!/bin/sh
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

INDEX_HTML="/usr/share/nginx/html/index.html"
TMP_INDEX_HTML="/tmp/index.html"
OLD_INDEX_HTML="/tmp/old-index.html"
# shellcheck disable=SC2155 # Used by index.html
export WEBSTATUS_VERSION="$(echo "$K_REVISION" | basenc --base64url)"
envsubst < "${INDEX_HTML}" > "${TMP_INDEX_HTML}"
cp "${INDEX_HTML}" "${OLD_INDEX_HTML}"
cp "${TMP_INDEX_HTML}" "${INDEX_HTML}"

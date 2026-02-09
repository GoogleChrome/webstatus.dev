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

FROM fsouza/fake-gcs-server:1.53.1

# Create a test bucket and land a sample file in there.
RUN mkdir -p /data/state-bucket && touch /data/state-bucket/sample.txt

# https://cloud.google.com/datastore/docs/tools/datastore-emulator
CMD ["-scheme", "http", "-public-host", "gcs:4443", "-external-url", "http://gcs:4443"]

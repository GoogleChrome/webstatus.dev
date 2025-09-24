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

FROM google/cloud-sdk:539.0.0-emulators

# https://cloud.google.com/datastore/docs/tools/datastore-emulator
ENTRYPOINT [ "gcloud", "beta", "emulators", "datastore", "start", "--host-port=0.0.0.0:8086", "--project=local", "--log-http", "--verbosity=debug", "--use-firestore-in-datastore-mode" ]

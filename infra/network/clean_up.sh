#!/bin/bash
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


# There may be some network resources that are enforced at the organization
# level. As a result, terraform will be unable to delete the network
# successfully. This script does a cleanup of any of those resources so that the
# network can be cleaned up successfully.

PROJECT_ID=$1
NETWORK_NAME=$2

gcloud compute firewall-rules delete --project="${PROJECT_ID}" \
    "$(gcloud compute firewall-rules list --project "${PROJECT_ID}" --filter="name~'${NETWORK_NAME}-*'" --format="value(name)")"
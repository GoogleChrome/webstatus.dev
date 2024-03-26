#!/bin/bash
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


# Check for required arguments
if [ $# -lt 5 ]; then
    echo "Usage: run_job.sh <job_image> <job_dockerfile> <job_service_dir> <job_yaml> <job_name>"
    exit 1
fi

job_image=$1
job_dockerfile=$2
job_service_dir=$3
job_yaml=$4
job_name=$5

set -ex
eval "$(minikube docker-env)"

# Build and push image
docker build -t "$job_image" --build-arg=service_dir="${job_service_dir}" --build-arg=MAIN_BINARY="job" -f "$job_dockerfile" .


# Cleanup any existing job with the same name
kubectl delete job "$job_name" --ignore-not-found=true

# Deploy the Job.
kubectl apply -f "$job_yaml"

# Wait for Job completion
kubectl wait --for=condition=complete --timeout=90s job/"$job_name"

# Get Job pod name
pod_name=$(kubectl get pods --selector=job-name="$job_name" -o jsonpath='{.items[0].metadata.name}')

# Get exit code
exit_code=$(kubectl get pods "$pod_name" -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}')

# Fetch logs
kubectl logs "$pod_name"

# Clean up
kubectl delete job "$job_name" --ignore-not-found=true

# Exit with the Job's exit code
exit "$exit_code"

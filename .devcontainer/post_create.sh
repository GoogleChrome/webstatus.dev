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

# Set umask inside the container for new files.
umask 022

# Make sure all files have read everyone for existing files.
chmod -R a+r ./*

# Clean up minikube just in case to ensure a fresh cluster.
make minikube-delete

# The mounted ~/.cache/go-build directory in .devcontainer.json is owned
# correctly but the parent ~/.cache directory is owned by root. This fixes that.
sudo chown "$(whoami)":"$(whoami)" ~/.cache/

# Install oapi-codegen
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1

# Install wrench CLI
go install github.com/cloudspannerecosystem/wrench@v1.11.8

# Install addlicense
go install github.com/google/addlicense@v1.1.1

# Install repo-wide npm tools
npm i --workspaces=false

# Generate files
make gen -B

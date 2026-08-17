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

# Ensure the mounted APT and user cache directories have correct ownership
# and configure APT to retain downloaded packages across DevContainer runs.
sudo mkdir -p /var/cache/apt/archives/partial
sudo chown -R "$(whoami)":"$(whoami)" /var/cache/apt/archives
sudo mkdir -p "$HOME/.cache/ms-playwright" "$HOME/.cache/go-build"
sudo chown -R "$(whoami)":"$(whoami)" "$HOME/.cache"
echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' | sudo tee /etc/apt/apt.conf.d/01keep-cache > /dev/null
sudo rm -f /etc/apt/apt.conf.d/docker-clean

# Install go tools
make go-install-tools

# Install repo-wide npm tools only (e.g. Playwright, gts)
npm ci --workspaces=false

# Generate files
make gen -B

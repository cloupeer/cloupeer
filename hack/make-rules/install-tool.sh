#!/usr/bin/env bash

# Copyright 2025 The Cloupeer Authors.
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

# ==============================================================================
# install-tool.sh
#
# This script provides a generic way to install versioned build tools.
# It is called by the main Makefile's file-based dependency rules.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "${PROJECT_ROOT}/hack/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - TOOLS_DIR:        The directory where local tools are installed.
#   - OPM_VERSION:      The specific version for the opm tool download URL.
# ==============================================================================

readonly TOOLS_DIR="${TOOLS_DIR:-${PROJECT_ROOT}/bin}"
readonly OPM_VERSION="${OPM_VERSION:-1.23.0}"

# ---
# Main Logic
# ---

# This script expects a single argument: the versioned filename to be created.
if [[ $# -ne 1 ]]; then
    error "Usage: $0 <versioned-tool-filename>"
    error "Example: $0 controller-gen-v0.16.1"
fi

# Ensure the target directory for tools exists.
mkdir -p "${TOOLS_DIR}"

readonly FULL_FILENAME="$1" # e.g., "controller-gen-v0.16.1"

# Robustly parse the tool name and version from the filename.
# The '.*' is greedy and will match up to the *last* '-v'.
readonly TOOL_NAME=$(echo "${FULL_FILENAME}" | sed -E 's/(.*)-v([^-]+$)/\1/')
readonly TOOL_VERSION=$(echo "${FULL_FILENAME}" | sed -E 's/(.*)-v([^-]+$)/\2/')
readonly TARGET_FILE="${TOOLS_DIR}/${FULL_FILENAME}"

# --- Idempotency Check ---
# If the target file already exists, do nothing.
if [ -f "${TARGET_FILE}" ]; then
    info "Tool '${TOOL_NAME}' version '${TOOL_VERSION}' is already installed. Skipping."
    exit 0
fi

info "Installing ${TOOL_NAME} v${TOOL_VERSION} to ${TARGET_FILE}..."

# Reliably determine the GOBIN path for 'go install'.
gobin=$(go env GOBIN)
[[ -z "$gobin" ]] && gobin="$(go env GOPATH)/bin"

case "$TOOL_NAME" in
    kustomize)
        go install "sigs.k.io/kustomize/kustomize/v5@v${TOOL_VERSION}"
        mv "${gobin}/kustomize" "${TARGET_FILE}"
        ;;
    controller-gen)
        go install "sigs.k8s.io/controller-tools/cmd/controller-gen@v${TOOL_VERSION}"
        mv "${gobin}/controller-gen" "${TARGET_FILE}"
        ;;
    setup-envtest)
        go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
        mv "${gobin}/setup-envtest" "${TARGET_FILE}"
        ;;
    golangci-lint)
        go install "github.com/golangci/golangci-lint/cmd/golangci-lint@v${TOOL_VERSION}"
        mv "${gobin}/golangci-lint" "${TARGET_FILE}"
        ;;
    operator-sdk)
        OS=$(go env GOOS)
        ARCH=$(go env GOARCH)
        URL="https://github.com/operator-framework/operator-sdk/releases/download/v${TOOL_VERSION}/operator-sdk_${OS}_${ARCH}"
        curl -sSLo "${TARGET_FILE}" "${URL}"
        chmod +x "${TARGET_FILE}"
        ;;
    opm)
        OS=$(go env GOOS)
        ARCH=$(go env GOARCH)
        # OPM's release tag might differ from the binary's version string, so we use the explicit version from config.mk.
        URL="https://github.com/operator-framework/operator-registry/releases/download/v${OPM_VERSION}/${OS}-${ARCH}-opm"
        curl -sSLo "${TARGET_FILE}" "${URL}"
        chmod +x "${TARGET_FILE}"
        ;;
    *)
        error "Unknown tool to install: ${TOOL_NAME}"
        ;;
esac

info "   -> Installed successfully."
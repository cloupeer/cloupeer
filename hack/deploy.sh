#!/usr/bin/env bash

# Copyright 2025 The Anankix Authors.
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
# deploy.sh
#
# This script is responsible for all deployment-related tasks that interact
# with a live Kubernetes cluster, such as installing CRDs and deploying
# specific controller manager components.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - KUSTOMIZE:        The path to the kustomize binary.
#   - KUBECTL:          The command to use for kubectl.
#   - IMG:              Optional. The full tag for the controller manager image.
#   - IMAGE_TAG_BASE:   The base for container images.
#   - VERSION:          The project version.
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly KUSTOMIZE="${KUSTOMIZE:-${PROJECT_ROOT}/bin/kustomize}"
readonly KUBECTL="${KUBECTL:-kubectl}"
readonly IMAGE_TAG_BASE="${IMAGE_TAG_BASE:-anankix.io/anankix}"
readonly VERSION="${VERSION:-dev}"

# ---
# Task Functions
# ---

# run_install applies all CRD manifests to the cluster.
# This is a global action for the entire project.
run_install() {
    info "Installing all CRDs into the cluster..."
    cd "${PROJECT_ROOT}"
    "${KUSTOMIZE}" build config/crd | "${KUBECTL}" apply -f -
}

# run_uninstall deletes all CRD manifests from the cluster.
run_uninstall() {
    info "Uninstalling all CRDs from the cluster..."
    cd "${PROJECT_ROOT}"
    # Consumes 'ignore-not-found' from the environment, defaulting to false.
    # Usage: make uninstall ignore-not-found=true
    local ignore_not_found="${ignore_not_found:-false}"
    "${KUSTOMIZE}" build config/crd | "${KUBECTL}" delete --ignore-not-found="${ignore_not_found}" -f -
}

# run_deploy deploys a specific component's controller manager to the cluster.
run_deploy() {
    local component_name="$1"
    info "Deploying controller manager for component '${component_name}'..."
    cd "${PROJECT_ROOT}"

    local kustomize_path="config/components/${component_name}"
    if ! [ -d "${kustomize_path}" ]; then
        error "Kustomize directory not found for component '${component_name}' at ${kustomize_path}"
    fi

    local img_tag="${IMG:-${IMAGE_TAG_BASE}-${component_name}:v${VERSION}}"
    info "    Setting image to: ${img_tag}"
    
    # Set the correct image for the deployment using kustomize.
    (cd "${kustomize_path}" && "${KUSTOMIZE}" edit set image controller="${img_tag}")

    # Build the final manifests from the component's overlay and apply them.
    "${KUSTOMIZE}" build "${kustomize_path}" | "${KUBECTL}" apply -f -
}

# run_undeploy removes a specific component's controller manager from the cluster.
run_undeploy() {
    local component_name="$1"
    info "Undeploying controller manager for component '${component_name}'..."
    cd "${PROJECT_ROOT}"
    
    local kustomize_path="config/components/${component_name}"
    if ! [ -d "${kustomize_path}" ]; then
        error "Kustomize directory not found for component '${component_name}' at ${kustomize_path}"
    fi

    local ignore_not_found="${ignore_not_found:-false}"
    "${KUSTOMIZE}" build "${kustomize_path}" | "${KUBECTL}" delete --ignore-not-found="${ignore_not_found}" -f -
}

# ---
# Main Dispatcher
# ---
main() {
    if [[ $# -eq 0 ]]; then
        error "No target specified for deploy.sh."
    fi

    local target="$1"
    local args=("${@:2}")

    info "Executing deployment target: ${target}"

    case "$target" in
        install)
            run_install
            ;;
        uninstall)
            run_uninstall
            ;;
        deploy)
            # We need a shared helper for this check. Let's assume it's in prelude.sh
            _require_one_component "$target" "$@"
            run_deploy "${args[0]}"
            ;;
        undeploy)
            _require_one_component "$target" "$@"
            run_undeploy "${args[0]}"
            ;;
        *)
            error "Unknown target '${target}' for deploy.sh."
            ;;
    esac
}

# ---
# Script Entrypoint
# ---
main "$@"

echo -e "\033[32m✅ Script 'deploy.sh' completed its task successfully.\033[0m"
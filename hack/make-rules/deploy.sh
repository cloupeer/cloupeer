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
#   - PUBLIC_REGISTRY:  The public registry for container images.
#   - VERSION:          The project version.
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly KUSTOMIZE="${KUSTOMIZE:-${PROJECT_ROOT}/bin/kustomize}"
readonly KUBECTL="${KUBECTL:-kubectl}"
readonly PUBLIC_REGISTRY="${PUBLIC_REGISTRY:-registry.cloupeer.io/cloupeer}"
readonly VERSION="${VERSION:-dev}"

# ---
# Task Functions
# ---

# run_install applies all base resources (CRDs, user-facing RBAC) to the cluster.
# This is a global action for the entire project.
run_install() {
    info "Installing all base resources into the cluster..."
    cd "${PROJECT_ROOT}"
    "${KUSTOMIZE}" build manifests/base | "${KUBECTL}" apply -f -
}

# run_uninstall deletes all base resources from the cluster.
run_uninstall() {
    info "Uninstalling all base resources from the cluster..."
    cd "${PROJECT_ROOT}"
    "${KUSTOMIZE}" build manifests/base | "${KUBECTL}" delete --ignore-not-found=true -f -
}

# run_deploy installs components to the cluster.
# It can install all components for an environment, or a single specific component.
run_deploy() {
    local env_name="$1"
    local component_name="$2"
    local kustomize_path

    cd "${PROJECT_ROOT}"

    if [[ "${component_name}" == "all" ]]; then
        info "Deploying all components for environment '${env_name}'..."
        info "    Setting images for all components..."
        for component in ${COMPONENTS}; do
            local component_kustomize_path="manifests/components/${component}/overlays/${env_name}"
            
            if ! [ -d "${component_kustomize_path}" ]; then
                error "Kustomize directory not found for component '${component}' at: ${component_kustomize_path}"
            fi

            local img_tag="${IMG:-${PUBLIC_REGISTRY}/${component}:v${VERSION}}"
            info "        - Setting image for '${component}' to: ${img_tag}"
            
            (cd "${component_kustomize_path}" && "${KUSTOMIZE}" edit set image "${component}"="${img_tag}")
        done
        
        kustomize_path="manifests/installation/${env_name}"

    else
        info "Deploying single component '${component_name}' for environment '${env_name}'..."
        kustomize_path="manifests/components/${component_name}/overlays/${env_name}"

        if ! [ -d "${kustomize_path}" ]; then
            error "Kustomize directory not found for component '${component_name}' at: ${kustomize_path}"
        fi

        local img_tag="${IMG:-${PUBLIC_REGISTRY}/${component_name}:v${VERSION}}"
        info "    Setting image to: ${img_tag}"
        
        (cd "${kustomize_path}" && "${KUSTOMIZE}" edit set image "${component_name}"="${img_tag}")
    fi

    info "    Applying manifests from: ${kustomize_path}"
    "${KUSTOMIZE}" build "${kustomize_path}" | "${KUBECTL}" apply -f -
    
    info "Successfully deployed '${component_name}' for environment '${env_name}'."
}

# run_undeploy uninstalls components from the cluster.
# It can uninstall all components for an environment, or a single specific component.
run_undeploy() {
    local env_name="$1"
    local component_name="$2"
    local kustomize_path

    cd "${PROJECT_ROOT}"

    if [[ "${component_name}" == "all" ]]; then
        info "Uninstalling all components from environment '${env_name}'..."
        kustomize_path="manifests/installation/${env_name}"
    else
        info "Uninstalling single component '${component_name}' from environment '${env_name}'..."
        kustomize_path="manifests/components/${component_name}/overlays/${env_name}"
    fi

    if ! [ -d "${kustomize_path}" ]; then
        error "Kustomize directory not found for undeploy at: ${kustomize_path}. Skipping."
    fi

    info "    Deleting manifests from: ${kustomize_path}"

    "${KUSTOMIZE}" build "${kustomize_path}" | "${KUBECTL}" delete --ignore-not-found=true -f -
    
    info "Successfully uninstalled '${component_name}' from environment '${env_name}'."
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
            _require_env_components "$target" "${args[@]}"
            run_deploy "${args[@]}"
            ;;
        undeploy)
            _require_env_components "$target" "${args[@]}"
            run_undeploy "${args[@]}"
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
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
# WITHOUT WARRANTIES OR CONDITIONS OF ANY, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ==============================================================================
# build.sh
#
# This script is the single entrypoint for all build-related tasks.
# It handles compiling Go binaries, running controllers locally, and building
# or pushing single-component or multi-arch container images.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - COMPONENTS:       A space-separated list of all components.
#   - VERSION:          The project version, used for embedding in the binary.
#   - PUSH_REGISTRY:    The registry for container images (e.g., ghcr.io/cloupeer).
#   - PLATFORMS:        Comma-separated list of platforms for multi-arch builds.
#   - CONTAINER_TOOL:   The container tool to use (docker, podman).
#   - IMG:              Optional. If set, overrides the calculated image tag.
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly COMPONENTS="${COMPONENTS:-}"
readonly VERSION="${VERSION:-dev}"
readonly PUSH_REGISTRY="${PUSH_REGISTRY:-ghcr.io/cloupeer}"
readonly PLATFORMS="${PLATFORMS:-linux/amd64}"
readonly CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
readonly OUTPUT_DIR="${OUTPUT_DIR:-${PROJECT_ROOT}/_output}"


# ---
# Task Functions
# ---

# build_binary compiles a single Go binary for a given component.
build_binary() {
    local component_name="$1"
    info "Building binary for component: ${component_name} (version: ${VERSION})"
    
    local output_path="${OUTPUT_DIR}/bin/${component_name%-cli}"
    local main_path="${PROJECT_ROOT}/cmd/${component_name}/main.go"

    if ! [ -f "${main_path}" ]; then
        error "Component main file not found at ${main_path}"
    fi

    # Embed version information into the binary using -ldflags.
    # Requires a 'version' variable in the main package.
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-X 'main.version=${VERSION}'" \
        -o "${output_path}" \
        "${main_path}"
}

# run_controller runs a single component controller locally.
run_controller() {
    local component_name="$1"
    info "Running component '${component_name}' locally..."
    go run "${PROJECT_ROOT}/cmd/${component_name}/main.go"
}

# build_docker_image builds a container image for a single component.
build_docker_image() {
    local component_name="$1"
    local img_tag="${IMG:-${PUSH_REGISTRY}/${component_name%-cli}:v${VERSION}}"

    info "Building container image for component '${component_name}': ${img_tag}"
    
    local dockerfile_path="${OUTPUT_DIR}/images/${component_name}/Dockerfile"
    if ! [ -f "${dockerfile_path}" ]; then
        error "Dockerfile not found at ${dockerfile_path}. Please run 'make dockerfiles' first."
    fi

    "${CONTAINER_TOOL}" build -f "${dockerfile_path}" -t "${img_tag}" .
}

# push_docker_image pushes a container image for a single component.
push_docker_image() {
    local component_name="$1"
    local img_tag="${IMG:-${PUSH_REGISTRY}/${component_name%-cli}:v${VERSION}}"
    info "Pushing container image: ${img_tag}"
    "${CONTAINER_TOOL}" push "${img_tag}"
}

# buildx_docker_image builds and pushes a multi-arch container image.
buildx_docker_image() {
    local component_name="$1"
    local img_tag="${IMG:-${PUSH_REGISTRY}/${component_name%-cli}:v${VERSION}}"

    info "Building and pushing multi-arch image for '${component_name}': ${img_tag}"
    info "Platforms: ${PLATFORMS}"

    local dockerfile_path="${OUTPUT_DIR}/images/${component_name}/Dockerfile"
    if ! [ -f "${dockerfile_path}" ]; then
        error "Dockerfile not found at ${dockerfile_path}. Please run 'make dockerfiles' first."
    fi

    local builder_name="cloupeer-builder"
    # Create and use a buildx builder if it doesn't already exist.
    if ! ${CONTAINER_TOOL} buildx ls | grep -q "${builder_name}"; then
        ${CONTAINER_TOOL} buildx create --name "${builder_name}" --use
    fi
    
    ${CONTAINER_TOOL} buildx build \
        --platform "${PLATFORMS}" \
        --tag "${img_tag}" \
        -f "${dockerfile_path}" \
        --push \
        .
    
    info "Multi-arch image build and push complete."
}


# ---
# Main Dispatcher
# ---
main() {
    if [[ $# -eq 0 ]]; then
        error "No target specified for build.sh. Please run via 'make <target>'."
    fi

    local target="$1"
    local args=("${@:2}") # Store all subsequent arguments in an array

    info "Executing build target: ${target}"
    
    case "$target" in
        build)
            local components_to_build
            if [ ${#args[@]} -eq 0 ]; then
                if [[ -z "${COMPONENTS}" ]]; then
                    error "No components to build. Check 'COMPONENTS' in build/config.mk"
                fi
                info "Building all components defined in config.mk..."
                # Convert space-separated string to array
                read -r -a components_to_build <<< "${COMPONENTS}"
            else
                info "Building specified components: ${args[*]}"
                components_to_build=("${args[@]}")
            fi
            
            for comp in "${components_to_build[@]}"; do
                build_binary "${comp}"
            done
            ;;

        run)
            _require_one_component "$target" "${args[@]}"
            run_controller "${args[0]}"
            ;;

        docker-build)
            _require_one_component "$target" "${args[@]}"
            build_docker_image "${args[0]}"
            ;;

        docker-push)
            _require_one_component "$target" "${args[@]}"
            push_docker_image "${args[0]}"
            ;;
            
        docker-buildx)
            _require_one_component "$target" "${args[@]}"
            buildx_docker_image "${args[0]}"
            ;;
        
        *)
            error "Unknown target '${target}' for build.sh."
            ;;
    esac
}

# ---
# Script Entrypoint
# ---
main "$@"

echo -e "\033[32m✅ Script 'build.sh' completed its task successfully.\033[0m"
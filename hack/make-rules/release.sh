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
# release.sh
#
# This script is responsible for all release-related tasks, such as creating
# distributable artifacts like a static install.yaml and OLM bundles/catalogs.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "${PROJECT_ROOT}/hack/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - KUSTOMIZE, OPM, IMG, BUNDLE_IMG, CATALOG_IMG, VERSION,
#   - CONTAINER_TOOL, PUBLIC_REGISTRY
#   - Optional: CHANNELS, DEFAULT_CHANNEL, USE_IMAGE_DIGESTS, BUNDLE_IMGS,
#     CATALOG_BASE_IMG
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly KUSTOMIZE="${KUSTOMIZE:-${PROJECT_ROOT}/bin/kustomize}"
readonly OPM="${OPM:-${PROJECT_ROOT}/bin/opm}"
readonly VERSION="${VERSION:-dev}"
readonly CONTAINER_TOOL="${CONTAINER_TOOL:-docker}"
readonly PUBLIC_REGISTRY="${PUBLIC_REGISTRY:-registry.cloupeer.io/cloupeer}"

# ---
# Task Functions
# ---

# generate_installer_yaml creates a single, static YAML file for a component.
generate_installer_yaml() {
    local component_name="$1"
    info "Generating consolidated installer YAML for component '${component_name}'..."
    
    local output_dir="${PROJECT_ROOT}/_output/release"
    mkdir -p "${output_dir}"
    local output_file="${output_dir}/${component_name}-install.yaml"
    
    local kustomize_path="manifests/components/${component_name}/overlays/prod"
    local img_tag="${IMG:-${PUBLIC_REGISTRY}/${component_name}:v${VERSION}}"

    (cd "${kustomize_path}" && "${KUSTOMIZE}" edit set image ${component_name}="${img_tag}")
    "${KUSTOMIZE}" build "${kustomize_path}" > "${output_file}"
    
    info "    --> Consolidated manifest written to ${output_file}"
}

# generate_bundle creates the OLM bundle manifests for a specific component.
generate_bundle() {
    local component_name="$1"
    info "Generating OLM bundle for component '${component_name}' (version ${VERSION})..."

    local output_dir="${PROJECT_ROOT}/_output/bundles/${component_name}"
    local kustomize_path="manifests/components/${component_name}/manifests"
    local img_tag="${IMG:-${PUBLIC_REGISTRY}/${component_name}:v${VERSION}}"

    # Dynamically build the bundle generation flags.
    local bundle_gen_flags="-q --overwrite --version ${VERSION} --package=${component_name} --output-dir=${output_dir}"
    if [[ -n "${CHANNELS:-}" ]]; then bundle_gen_flags+=" --channels=${CHANNELS}"; fi
    if [[ -n "${DEFAULT_CHANNEL:-}" ]]; then bundle_gen_flags+=" --default-channel=${DEFAULT_CHANNEL}"; fi
    if [[ "${USE_IMAGE_DIGESTS:-false}" == "true" ]]; then bundle_gen_flags+=" --use-image-digests"; fi

    # Generate manifests using the correct image, then generate and validate the bundle.
    (cd "${kustomize_path}" && "${KUSTOMIZE}" edit set image ${component_name}="${img_tag}")
    "${KUSTOMIZE}" build "${kustomize_path}" | "${OPERATOR_SDK}" generate bundle ${bundle_gen_flags}
    "${OPERATOR_SDK}" bundle validate "${output_dir}"
    
    info "    --> OLM Bundle for '${component_name}' generated and validated in ${output_dir}"
}

# build_bundle_image builds the container image for a component's OLM bundle.
build_bundle_image() {
    local component_name="$1"
    local bundle_img_tag="${BUNDLE_IMG:-${PUBLIC_REGISTRY}/${component_name}-bundle:v${VERSION}}"
    info "Building OLM bundle image for '${component_name}': ${bundle_img_tag}"
    
    local bundle_root="${PROJECT_ROOT}/_output/bundles/${component_name}"
    
    ${CONTAINER_TOOL} build -f "${bundle_root}/bundle.Dockerfile" -t "${bundle_img_tag}" "${bundle_root}"
}

# push_bundle_image pushes a component's OLM bundle image.
push_bundle_image() {
    local component_name="$1"
    local bundle_img_tag="${BUNDLE_IMG:-${PUBLIC_REGISTRY}/${component_name}-bundle:v${VERSION}}"
    info "Pushing OLM bundle image: ${bundle_img_tag}"
    ${CONTAINER_TOOL} push "${bundle_img_tag}"
}

# build_catalog_image builds an OLM catalog image, potentially containing multiple bundles.
build_catalog_image() {
    local catalog_img_tag="${CATALOG_IMG:-${PUBLIC_REGISTRY}-catalog:v${VERSION}}"
    info "Building OLM catalog image: ${catalog_img_tag}"
    
    # BUNDLE_IMGS is a comma-separated list of bundle images to include.
    # If not provided, it's an error because a catalog needs content.
    if [[ -z "${BUNDLE_IMGS:-}" ]]; then
        error "BUNDLE_IMGS environment variable must be set. \nUsage: make catalog-build BUNDLE_IMGS=<image1>,<image2>"
    fi
    
    local from_index_opt=""
    if [[ -n "${CATALOG_BASE_IMG:-}" ]]; then
        from_index_opt="--from-index ${CATALOG_BASE_IMG}"
    fi

    "${OPM}" index add \
        --container-tool "${CONTAINER_TOOL}" \
        --mode semver \
        --tag "${catalog_img_tag}" \
        --bundles "${BUNDLE_IMGS}" \
        ${from_index_opt}
}

# push_catalog_image pushes the OLM catalog image.
push_catalog_image() {
    local catalog_img_tag="${CATALOG_IMG:-${PUBLIC_REGISTRY}-catalog:v${VERSION}}"
    info "Pushing OLM catalog image: ${catalog_img_tag}"
    ${CONTAINER_TOOL} push "${catalog_img_tag}"
}


# ---
# Main Dispatcher
# ---
main() {
    if [[ $# -eq 0 ]]; then
        error "No target specified for release.sh."
    fi

    local target="$1"
    local args=("${@:2}")
    info "Executing release target: ${target}"

    case "$target" in
        installer)
            _require_one_component "$target" "${args[@]}"
            generate_installer_yaml "${args[0]}"
            ;;
        bundle)
            _require_one_component "$target" "${args[@]}"
            generate_bundle "${args[0]}"
            ;;
        bundle-build)
            _require_one_component "$target" "${args[@]}"
            build_bundle_image "${args[0]}"
            ;;
        bundle-push)
            _require_one_component "$target" "${args[@]}"
            push_bundle_image "${args[0]}"
            ;;
        catalog-build)
            build_catalog_image
            ;;
        catalog-push)
            push_catalog_image
            ;;
        *)
            error "Unknown target '${target}' for release.sh."
            ;;
    esac
}

# --- 
# Script Entrypoint 
# ---
main "$@"

echo -e "\033[32m✅ Script 'release.sh' completed its task successfully.\033[0m"
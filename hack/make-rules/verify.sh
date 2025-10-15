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
# verify.sh
#
# This script is responsible for all code verification tasks,
# such as formatting, vetting, and linting.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "${PROJECT_ROOT}/hack/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - GOLANGCI_LINT
#   - COMPONENTS
#   - COMPONENT_PATH_MAP
#   - COMMON_PACKAGE_SCOPE
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly GOLANGCI_LINT="${GOLANGCI_LINT:-${PROJECT_ROOT}/bin/golangci-lint}"
readonly COMPONENT_PATH_MAP="${COMPONENT_PATH_MAP:-}"
readonly COMMON_PACKAGE_SCOPE="${COMMON_PACKAGE_SCOPE:-./api/...}"

# _get_packages_for_component resolves all relevant Go packages for a given set
# of components. It accepts multiple component names as arguments and returns a
# unique, space-separated list of package paths.
_get_packages_for_components() {
    # 1. Initialize an array to accumulate all found package paths.
    local all_packages=()

    # 2. Loop through all component names passed as arguments to the function.
    for comp in "$@"; do
        # Add the component's cmd directory.
        all_packages+=("./cmd/${comp}/...")

        # Resolve the 'internal' path using the map, with a fallback to the component name.
        local internal_path=""
        local mapping_found=false
        # Rule 1: Check for a direct mapping in COMPONENT_PATH_MAP. This has the highest priority.
        for mapping in ${COMPONENT_PATH_MAP}; do
            local cmd_name="${mapping%%:*}"  # Get the part before the colon
            local internal_name="${mapping##*:}" # Get the part after the colon
            if [[ "${cmd_name}" == "${comp}" ]]; then
                internal_path="${internal_name}"
                mapping_found=true
                break
            fi
        done

        # If no mapping was found, apply the fallback naming convention rules.
        if ! ${mapping_found}; then
            if [[ "${comp}" == "cpeer-"* ]]; then
                # Rule 2: Component name has the "cpeer-" prefix.
                # Remove the "cpeer-" prefix. e.g., "cpeer-edge-agent" -> "edge-agent"
                local path_without_prefix="${comp#cpeer-}"
                # Remove all remaining hyphens. e.g., "edge-agent" -> "edgeagent"
                internal_path="${path_without_prefix//-/}"
            else
                # Rule 3: Component name does not have the "cpeer-" prefix.
                # Use the component name directly. e.g., "cpeerctl" -> "cpeerctl"
                internal_path="${comp}"
            fi
        fi

        # Find matching directories within './internal'.
        # Using 'mapfile' (or 'readarray') is a safer way to read find's output
        # into an array, correctly handling paths with special characters.
        local found_paths
        mapfile -t found_paths < <(find ./internal -type d -name "${internal_path}" 2>/dev/null)

        if [[ ${#found_paths[@]} -gt 0 ]]; then
            for path in "${found_paths[@]}"; do
                all_packages+=("${path}/...")
            done
        else
            echo "Warning: No internal package found for component '${comp}' (resolved internal name: '${internal_path}')" >&2
        fi
    done

    # 3. Combine the discovered packages with the common packages, then de-duplicate.
    local final_packages
    # The tr/sort/tr pipeline is a robust way to get a unique list of space-separated items.
    final_packages=$(echo "${all_packages[@]}" "${COMMON_PACKAGE_SCOPE[@]}" | tr ' ' '\n' | sort -u | tr '\n' ' ')

    # 4. Return the final, clean list of packages.
    echo "${final_packages}"
}


# ---
# Task Functions
# ---

# run_fmt formats all Go code in the project.
run_fmt() {
    info "Formatting all Go code..."
    go fmt ./...
}

# run_vet runs 'go vet' on all packages to catch subtle errors.
run_vet() {
    info "Running go vet on all packages..."
    go vet ./...
}

# run_lint runs the golangci-lint linter against a given list of packages.
run_lint() {
    info "Running golangci-lint..."
    local packages_to_check=("$@")

    if ! [ -f "${GOLANGCI_LINT}" ]; then
        error "golangci-lint not found. Please run 'make lint' to ensure it is installed."
    fi

    "${GOLANGCI_LINT}" run "${packages_to_check[@]}"
}

# run_lint_fix runs golangci-lint with the --fix flag.
run_lint_fix() {
    info "Running golangci-lint with --fix..."
    local packages_to_check=("$@")

    if ! [ -f "${GOLANGCI_LINT}" ]; then
        error "golangci-lint not found. Please run 'make lint-fix' to ensure it is installed."
    fi

    "${GOLANGCI_LINT}" run --fix "${packages_to_check[@]}"
}


# ---
# Main Dispatcher
# ---
main() {
    if [[ $# -eq 0 ]]; then
        error "No target specified for verify.sh."
    fi

    local target="$1"
    local components=("${@:2}")

    info "Executing verify target: ${target}"

    case "$target" in
        fmt)
            # 'fmt' is fast and always runs on the entire project.
            run_fmt
            ;;
        vet)
            # 'vet' is also fast and always runs on the entire project.
            run_vet
            ;;
        lint | lint-fix)
            # 'lint' can be slow, so we support scoping it to specific components.
            local packages_to_check=()
            if [ ${#components[@]} -eq 0 ]; then
                info "Scope: All packages"
                packages_to_check=("./...")
            else
                info "Scope: Specified components (${components[*]})"
                # The improved function handles all components at once, so we can
                # replace the for-loop with a single, clean function call.
                # The result is captured into an array.
                packages_to_check=($(_get_packages_for_components "${components[@]}"))
            fi

            info "==============================="
            info "Packages to check: ${packages_to_check[*]}"
            info "==============================="

            if [[ "$target" == "lint-fix" ]]; then
                run_lint_fix "${packages_to_check[@]}"
            else
                run_lint "${packages_to_check[@]}"
            fi
            ;;
        *)
            error "Unknown target '${target}' for verify.sh."
            ;;
    esac
}

# ---
# Script Entrypoint
# ---
main "$@"

echo -e "\033[32m✅ Script 'verify.sh' completed its task successfully.\033[0m"
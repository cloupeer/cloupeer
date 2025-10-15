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
# test.sh
#
# This script is responsible for running unit, integration, and e2e tests.
# It can run tests for all components or for a specified subset.
# ==============================================================================

# Source the common prelude script to set up the environment and helpers.
# shellcheck source=lib/prelude.sh
source "${PROJECT_ROOT}/hack/lib/prelude.sh"

# ==============================================================================
# Consumed Environment Variables (from build/config.mk)
# ------------------------------------------------------------------------------
#   - TOOLS_DIR:            The directory where local tools are installed.
#   - OUTPUT_DIR:           The directory for all build artifacts.
#   - ENVTEST:              The path to the setup-envtest binary.
#   - ENVTEST_K8S_VERSION:  The version of Kubernetes to use for envtest.
# ==============================================================================

# Provide default values for consumed environment variables for robustness.
readonly TOOLS_DIR="${TOOLS_DIR:-${PROJECT_ROOT}/bin}"
readonly OUTPUT_DIR="${OUTPUT_DIR:-${PROJECT_ROOT}/_output}"
readonly ENVTEST="${ENVTEST:-${TOOLS_DIR}/setup-envtest}"
readonly ENVTEST_K8S_VERSION="${ENVTEST_K8S_VERSION:-1.31.0}"

# ---
# Task Functions
# ---

# run_unit_tests runs unit and integration tests for a given list of packages.
run_unit_tests() {
    info "Setting up envtest assets for Kubernetes ${ENVTEST_K8S_VERSION}..."
    
    # Ensure envtest assets are downloaded and export the path for 'go test'.
    export KUBEBUILDER_ASSETS=$("${ENVTEST}" use "${ENVTEST_K8S_VERSION}" --bin-dir "${TOOLS_DIR}" -p path)
    if [[ -z "$KUBEBUILDER_ASSETS" ]]; then
        error "Failed to get KUBEBUILDER_ASSETS path from envtest."
    fi
    info "KUBEBUILDER_ASSETS is set to: ${KUBEBUILDER_ASSETS}"

    info "Running unit and integration tests..."
    
    # The first argument is the list of packages to test.
    local packages_to_test=("$@")

    # Ensure the output directory for the coverage report exists.
    mkdir -p "${OUTPUT_DIR}"
    go test "${packages_to_test[@]}" -coverprofile "${OUTPUT_DIR}/cover.out"
}

# run_e2e_tests runs end-to-end tests for a given list of packages.
run_e2e_tests() {
    info "Running end-to-end (e2e) tests..."
    # NOTE: e2e tests assume a live Kubernetes cluster is configured via KUBECONFIG.
    
    local packages_to_test=("$@")
    go test "${packages_to_test[@]}" -v -ginkgo.v
}


# ---
# Main Dispatcher
# ---
main() {
    if [[ $# -eq 0 ]]; then
        error "No test type specified for test.sh. Must be 'unit' or 'e2e'."
    fi

    local test_type="$1"
    local components=("${@:2}")
    local packages_to_test=() # Initialize an empty array for package paths

    info "Executing test type: ${test_type}"

    case "$test_type" in
        unit)
            if [ ${#components[@]} -eq 0 ]; then
                info "Preparing to test all components (excluding e2e)..."
                mapfile -t packages_to_test < <(go list ./... | grep -v /test/e2e)
            else
                info "Preparing to test specified components: ${components[*]}"
                for comp in "${components[@]}"; do
                    # Add all packages for a specific component. Customize paths as needed.
                    mapfile -t -O "${#packages_to_test[@]}" packages_to_test < <(go list ./cmd/${comp}/... ./internal/controller/${comp}/... ./api/... | grep -v /test/e2e)
                done
            fi
            
            if [ ${#packages_to_test[@]} -eq 0 ]; then
                info "No packages found to test. Exiting."
                exit 0
            fi

            run_unit_tests "${packages_to_test[@]}"
            ;;
        e2e)
            if [ ${#components[@]} -eq 0 ]; then
                info "Preparing to run all e2e tests..."
                packages_to_test=("./test/e2e/...")
            else
                info "Preparing to run e2e tests for specified components: ${components[*]}"
                 for comp in "${components[@]}"; do
                    packages_to_test+=("./test/e2e/${comp}/...")
                done
            fi
            
            run_e2e_tests "${packages_to_test[@]}"
            ;;
        *)
            error "Unknown test type '${test_type}'."
            ;;
    esac
}


# ---
# Script Entrypoint
# ---
main "$@"

echo -e "\033[32m✅ Script 'test.sh' completed its task successfully.\033[0m"
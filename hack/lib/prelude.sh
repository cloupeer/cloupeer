#!/usr/bin/env bash

# Copyright 2025 The Autopeer Authors.
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
# prelude.sh
#
# This script is intended to be sourced by all other executable scripts.
# It sets up a standard, robust execution environment by:
#   - Enabling strict mode.
#   - Determining the project's root directory and changing into it.
#   - Providing common helper functions (info, error).
# ==============================================================================

# Enable strict mode (fail on errors, unbound variables, and pipe failures).
set -euo pipefail

# --- Environment Setup ---

# Robustly find the project root using git and change the current directory to it.
# This ensures that all subsequent script logic runs from a predictable location.
if ! git_root=$(git rev-parse --show-toplevel 2>/dev/null); then
    echo "ERROR: This script must be run from within a Git repository." >&2
    exit 1
fi
readonly PROJECT_ROOT="${git_root}"
cd "${PROJECT_ROOT}"


# --- Helper Functions ---

# info prints a green-colored message prefixed with '-->'.
# Usage: info "Doing something..."
info() {
    echo -e "\033[32m--> ${1}\033[0m"
}

# error prints a red-colored error message and exits with a non-zero status.
# Usage: error "Something went wrong."
error() {
    echo -e "\033[31mERROR: ${1}\033[0m" >&2
    exit 1
}


# --- Helper Functions ---

# _require_one_component ensures that exactly one component name is passed as an argument.
# This is a shared helper for scripts that operate on a single component.
# Usage: _require_one_component <target_name> <arg_array>
_require_one_component() {
    local target_name="$1"
    # Create an array from the second argument onwards
    local args=("${@:2}")

    if [[ ${#args[@]} -ne 1 ]] || [[ -z "${args[0]}" ]]; then
        error "Exactly one component name is required for the '${target_name}' target. Usage: 'make ${target_name} <component_name>'"
    fi
}

# _require_env_components ensures that exactly two non-empty arguments are provided:
# an environment name and a component name (or 'all').
#
# This is a dedicated validation function for targets like 'deploy' or 'undeploy'
# to verify they receive the correct number and type of arguments before proceeding.
#
# Parameters:
#   $1 - target_name: The name of the make target that called this function (e.g., "deploy").
#   $@ - args: The array of arguments passed to the make target (e.g., "development" "my-app").
_require_env_components() {
    local target_name="$1"
    local args=("${@:2}")

    if [[ ${#args[@]} -ne 2 ]] || [[ -z "${args[0]}" ]] || [[ -z "${args[1]}" ]]; then
        local error_message
        error_message=$(printf "Exactly one environment and one component name are required for the '%s' target.\nUsage: 'make %s <env_name> all' OR 'make %s <env_name> <component_name>'" \
            "${target_name}" "${target_name}" "${target_name}")
        
        error "${error_message}"
    fi
}

# _require_non_empty_args ensures that at least one argument is passed,
# and that ALL passed arguments are non-empty strings.
# Usage: _require_non_empty_args <target_name> <arg_array>
_require_non_empty_args() {
    local target_name="$1"
    local args=("${@:2}")

    if [[ ${#args[@]} -eq 0 ]]; then
        error "At least one argument is required for the '${target_name}' target. None were provided."
    fi

    for arg in "${args[@]}"; do
        if [[ -z "${arg}" ]]; then
            error "Arguments for the '${target_name}' target cannot be empty strings. Please provide valid arguments."
        fi
    done
}

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
# build/config.mk
#
# This file serves as the Single Source of Truth (SSoT) for all build
# configurations. All variables defined here are exported to the environment
# and are consumed by the scripts in the scripts/ directory.
# ==============================================================================


#@ Core Environment & Paths
# ------------------------------------------------------------------------------
# Use 'shell pwd' to ensure a real, non-symlinked path.
export PROJECT_ROOT   := $(shell pwd)
export TOOLS_DIR      := $(PROJECT_ROOT)/bin
export OUTPUT_DIR     := $(PROJECT_ROOT)/_output


#@ Project Identity & Version
# ------------------------------------------------------------------------------
# The base for all container images.
export PUBLIC_REGISTRY  := ghcr.io/autopeer
export PUSH_REGISTRY    := ghcr.io/autopeer
# The project version. Defaults to the output of 'git describe'.
export VERSION          ?= $(shell git describe --tags --always --dirty)


#@ Build Customization
# ------------------------------------------------------------------------------
# A space-separated list of specific components to build, e.g., BINS="manager comp2".
# If empty, all components will be built.
BINS ?=
# Target platforms for multi-arch container images.
export PLATFORMS ?= linux/amd64,linux/arm64
# The container tool to use for all image operations.
export CONTAINER_TOOL ?= docker


#@ Component Auto-Discovery & Mapping
# ------------------------------------------------------------------------------
# List of component directories under cmd/ to exclude from discovery.
export EXCLUDE_COMPONENTS :=
# Discover all component directories under cmd/ and apply the exclusion list.
ALL_COMPONENTS     := $(sort $(shell find cmd -mindepth 1 -maxdepth 1 -type d -exec basename {} \;))
export COMPONENTS  := $(filter-out $(EXCLUDE_COMPONENTS), $(ALL_COMPONENTS))

# Define the mapping between a component in 'cmd/' and its corresponding
# implementation directory in 'internal/'.
# Format: <cmd-component-name>:<internal-directory-name>
# If a component is not listed here, it is assumed its internal directory has the same name.
export COMPONENT_PATH_MAP := \
    cpeer-controller-manager:controller

# Define other common package paths that should always be included in a component's scope.
export COMMON_PACKAGE_SCOPE := ./api/...


#@ Tooling & Dependencies
# ------------------------------------------------------------------------------
# This section defines all external build-time dependencies.

# Versions
export GOLANG_VERSION           ?= 1.25.1  # The Go version for Autopeer
export ENVTEST_K8S_VERSION      ?= 1.31.10 # The K8s version for envtest assets

# Tool Versions
export KUSTOMIZE_VERSION        ?= 5.8.0
export CONTROLLER_GEN_VERSION   ?= 0.16.1
export ENVTEST_VERSION          ?= # Use a specific version number
export GOLANGCI_LINT_VERSION    ?= 1.64.8
export OPERATOR_SDK_VERSION     ?= 1.39.2
export OPM_VERSION              ?= 1.23.0
export HELM_VERSION             ?= 3.14.0


# Tool Paths
# These variables construct the full path to the versioned tool binaries.
export KUSTOMIZE        := $(TOOLS_DIR)/kustomize-v$(KUSTOMIZE_VERSION)
export CONTROLLER_GEN   := $(TOOLS_DIR)/controller-gen-v$(CONTROLLER_GEN_VERSION)
export ENVTEST          := $(TOOLS_DIR)/setup-envtest
export GOLANGCI_LINT    := $(TOOLS_DIR)/golangci-lint-v$(GOLANGCI_LINT_VERSION)
export OPERATOR_SDK     := $(TOOLS_DIR)/operator-sdk-v$(OPERATOR_SDK_VERSION)
export OPM              := $(TOOLS_DIR)/opm-v$(OPM_VERSION)
export HELM             := $(TOOLS_DIR)/helm-v$(HELM_VERSION)

# A consolidated list of all tool names, used for dependency management in the main Makefile.
TOOLS := kustomize controller-gen envtest golangci-lint operator-sdk opm


#@ Derived Image Tags
# ------------------------------------------------------------------------------
# These are the full image tags derived from the base name and version.
# They can be overridden on the command line if needed (e.g., make docker-push IMG=...).
export IMG          ?= 
export BUNDLE_IMG   ?= 
export CATALOG_IMG  ?= 


#@ Infrastructure Components
# ------------------------------------------------------------------------------
# List of components that should maintain static configurations.
# These components are exempt from dynamic version/image injection during deployment
# to prevent stateful workload restarts (e.g. StatefulSet forbidden updates).
export INFRA_COMPONENTS ?= emqx mysql redis prometheus minio

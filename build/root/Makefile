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

# ---
# Configuration
# ---

# Import the central configuration file. This serves as the Single Source of Truth (SSoT)
# for all user-configurable variables (versions, image names, etc.) used in the build system.
include build/config.mk

# ---
# Core Makefile Settings
#
# This section defines the fundamental behavior of the Makefile, including
# debugging, shell environment, and the default goal.
# ---

# Makefile Debugging:
# Define a debug flag. Run `make DBG_MAKEFILE=1 <target>` to enable verbose
# output for debugging the Makefile parsing and execution flow.
DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # In normal mode (default), enable the silent flag (-s) to prevent make from
    # echoing the commands being executed. This provides a much cleaner user output.
    MAKEFLAGS += -s
endif

# Shell Settings:
# Use bash as the shell for all recipes, and enforce a "strict mode".
# -o errexit:  Exit immediately if a command exits with a non-zero status.
# -o pipefail: The return value of a pipeline is the status of the last command
#              to exit with a non-zero status, or zero if no command failed.
# -o nounset:  Treat unset variables as an error when substituting.
# This creates a more robust and predictable build environment.
SHELL := /usr/bin/env bash -o errexit -o pipefail -o nounset

# Set default flags for the shell.
# -e: Equivalent to -o errexit.
# -c: Tells the shell to read commands from the following string.
# This is the standard way make invokes the shell for each recipe line.
.SHELLFLAGS = -ec

# Default Goal:
# Set the default target to 'help'. This means running 'make' without any
# arguments will automatically display the help message.
.DEFAULT_GOAL := help

# ---
# Command-line Argument Parsing ("Black Magic")
# ------------------------------------------------------------------------------
# This block enables a more user-friendly CLI experience by simulating the
# ability to pass arguments directly after a target, for example:
#   'make run manager'
# instead of the more verbose 'make run COMPONENT=manager'.
# ---

# 1. Capture the raw command-line goals (e.g., "run manager").
_all_goals := $(MAKECMDGOALS)

# 2. Define a "whitelist" of targets that are designed to accept a trailing argument.
_targets_with_args := run docker-build docker-push docker-buildx installer bundle bundle-build bundle-push deploy undeploy

# 3. Check if one of our whitelisted targets was actually invoked.
#    'filter' finds the intersection, and 'firstword' grabs the first one found.
_invoked_target := $(firstword $(filter $(_targets_with_args), $(_all_goals)))

# 4. If a whitelisted target was found, extract the argument(s) that come *after* it.
#    'filter-out' removes the known target from the goal list, leaving only the arguments.
component_arg := $(if $(_invoked_target),$(filter-out $(_invoked_target), $(_all_goals)))

# Solution: Dynamically declare all argument-like words as phony targets with an empty recipe.
# This prevents "No rule to make target..." errors for arguments passed after the main target.
# It correctly handles patterns like `make <target> <arg1>` and `make <target> <arg1> <arg2>`.
.PHONY: $(component_arg)
$(component_arg): ; # The semicolon is stylistic for an explicit empty recipe, and is optional.

# ---
# Dependency Management
#
# This section implements a generic and scalable system for managing the download
# and installation of versioned build tools listed in the $(TOOLS) variable.
# It ensures that tools are downloaded only when they are missing.
# ---

# Step 1: Dynamically create a user-friendly phony target for each tool.
# This allows developers to run `make controller-gen` as a convenient shortcut
# to ensure the tool is installed.
$(foreach tool,$(TOOLS),$(eval .PHONY: $(tool)))

# Step 2: Create a static pattern rule that links the simple phony target (e.g., "controller-gen")
# to its corresponding versioned file on disk (e.g., "bin/controller-gen-vX.Y.Z").
# This is the core of `make`'s dependency tracking. When `make controller-gen` is run,
# `make` sees that it depends on the actual file, and will then check if the file exists.
$(TOOLS): %: $(TOOLS_DIR)/%-v$($(shell echo $* | tr 'a-z-' 'A-Z_')_VERSION) ## Download the $* tool if necessary.
	@# This target's recipe is empty. Its sole purpose is to establish the dependency link.

# Step 3: Define a general pattern rule for how to actually create any tool file if it's missing.
# This rule is triggered by the dependency check in Step 2. If the file (e.g., "bin/controller-gen-vX.Y.Z")
# does not exist, this recipe will be executed.
$(TOOLS_DIR)/%:
	@# The recipe calls a shell script, passing the tool name and version extracted from the target filename.
	# For a target like "bin/controller-gen-v0.16.1", $(notdir $@) becomes "controller-gen-v0.16.1".
	@./hack/make-rules/install-tool.sh $(notdir $@)

# Dynamically create an individual target for each component's Dockerfile generation.
$(foreach comp,$(COMPONENTS),$(eval .PHONY: gen-dockerfile-$(comp)))
$(foreach comp,$(COMPONENTS),$(eval gen-dockerfile-$(comp): ; @./hack/make-rules/generate.sh dockerfile $(comp)))

# ---
# User-Facing Targets
# ---

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
.PHONY: help
help: ## Display this help.
	@# Part 1: Use awk to print all statically defined targets from the Makefiles.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

	@# Part 2: Dynamically print help for all tools defined in the $(TOOLS) list.
	@printf "\n\033[1m%s\033[0m\n" "Tooling & Dependencies"
	@$(foreach tool,$(TOOLS), \
		printf "  \033[36m%-20s\033[0m %s\n" "$(tool)" "Download the $(tool) tool if necessary."; \
	)


##@ Development

.PHONY: manifests-emqx
manifests-emqx: $(HELM) ## Generate K8s manifests for EMQX from its Helm chart.
	@echo "Adding EMQX Helm repository..."
	@$(HELM) repo add emqx https://repos.emqx.io/charts
	@$(HELM) repo update
	@echo "Generating EMQX manifests from Helm chart..."
	@mkdir -p manifests/components/emqx/base
	@$(HELM) template emqx emqx/emqx \
		--version 5.8.8 \
		--namespace autopeer-io \
		-f manifests/components/emqx/values.yaml \
		> manifests/components/emqx/base/emqx-generated.yaml
	@echo "EMQX manifests generated successfully."

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## Generate K8s manifests (CRDs, RBAC, etc.) in `config/`.
	@./hack/make-rules/generate.sh manifests

.PHONY: generate
generate: $(CONTROLLER_GEN) ## Generate deepcopy code (zz_generated.deepcopy.go).
	@./hack/make-rules/generate.sh generate

.PHONY: dockerfiles
dockerfiles: $(addprefix gen-dockerfile-,$(COMPONENTS)) ## Generate all component Dockerfiles in `_output/images/`.
	@# This is an aggregator target, the real work is in the dependencies.

.PHONY: protoc
protoc: 
	@protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/v1/hub.proto

.PHONY: fmt
fmt: ## Format Go code 
	@./hack/make-rules/verify.sh fmt

.PHONY: vet
vet: ## Run go vet against code.
	@./hack/make-rules/verify.sh vet

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint linter. Use 'make lint BINS="comp"' for a single one.
	@./hack/make-rules/verify.sh lint $(BINS)

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Run golangci-lint linter and perform fixes.
	@./hack/make-rules/verify.sh lint-fix $(BINS)

.PHONY: test
test: manifests generate fmt vet $(ENVTEST) ## Run unit and integration tests (excluding e2e).
	@./hack/make-rules/test.sh unit $(BINS)

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests (requires a running cluster).
	@./hack/make-rules/test.sh e2e $(BINS)


##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build all component binaries. All: 'make build'. Specific: 'make build BINS="comp1 comp2"'.
	@./hack/make-rules/build.sh build $(BINS)

.PHONY: run
run: manifests generate fmt vet ## Run a component. Usage: 'make run <component_name> ARGS="--arg1 val1"'
	@./hack/make-rules/build.sh run $(component_arg) $(ARGS)

.PHONY: docker-build
docker-build: dockerfiles ## Build a container image. Usage: 'make docker-build <component_name>'.
	@./hack/make-rules/build.sh docker-build $(component_arg)

.PHONY: docker-push
docker-push: ## Push a container image. Usage: 'make docker-push <component_name>'.
	@./hack/make-rules/build.sh docker-push $(component_arg)

.PHONY: docker-buildx
docker-buildx: dockerfiles ## Build and push a multi-arch container image. Usage: 'make docker-buildx <component_name>'.
	@./hack/make-rules/build.sh docker-buildx $(component_arg)


##@ Deployment

.PHONY: install
install: manifests $(KUSTOMIZE) ## Install CRDs into the K8s cluster.
	@./hack/make-rules/deploy.sh install

.PHONY: uninstall
uninstall: manifests $(KUSTOMIZE) ## Uninstall CRDs from the K8s cluster.
	@./hack/make-rules/deploy.sh uninstall

.PHONY: deploy
deploy: install $(KUSTOMIZE) ## Deploy controller to the K8s cluster.
	@./hack/make-rules/deploy.sh deploy $(component_arg)

.PHONY: undeploy
undeploy: $(KUSTOMIZE) ## Undeploy controller from the K8s cluster.
	@./hack/make-rules/deploy.sh undeploy $(component_arg)


##@ Release

.PHONY: installer
installer: manifests generate $(KUSTOMIZE) ## Generate a consolidated install.yaml in `_output/release/`.
	@./hack/make-rules/release.sh installer $(component_arg)

.PHONY: bundle
bundle: manifests generate $(KUSTOMIZE) ## Generate bundle manifests and metadata.
	@./hack/make-rules/release.sh bundle $(component_arg)

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	@./hack/make-rules/release.sh bundle-build $(component_arg)

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	@./hack/make-rules/release.sh bundle-push $(component_arg)

.PHONY: catalog-build
catalog-build: $(OPM) ## Build a catalog image.
	@./hack/make-rules/release.sh catalog-build

.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	@./hack/make-rules/release.sh catalog-push

VERSION ?= latest

build_dir     := target
functions_dir := functions
functions     := $(shell ls cmd)
sh_src        := $(shell find . -type f -name '*.sh')

# Variables consumed by scripts
export MAKE
export BUILD_DIR := $(build_dir)

no_color := \033[0m
ok_color := \033[38;5;74m

# Function for printing a pretty banner
banner = \
	echo "\n$(ok_color)=====> $1$(no_color)"

# Function for checking that a variable is defined
check_defined = \
	$(if $(value $1),,$(error Error: Variable $1 is required but undefined))

$(build_dir):
	@mkdir -p $(build_dir)

.PHONY: clean
clean:
	@$(call banner,Cleaning)
	rm -rf ./$(build_dir)

.PHONY: lint
lint:
	@$(call banner,Running Shfmt)
	@shfmt -i 2 -ci -sr -bn -d "$(sh_src)"
	@$(call banner,Running Shellcheck)
	@shellcheck "$(sh_src)"

.PHONY: test
test:
	@$(call banner,Running tests)
	@go test ./...

.PHONY: build-$(functions)
build-$(functions):
	@$(call banner,Building Kpt function gantry-$(functions):$(VERSION))
	docker build -t gantry-$(functions):$(VERSION) --build-arg FUNCTION=$(functions) .

.PHONY: publish-$(functions)
publish-$(functions): build-$(functions)
	@$(call banner,Publishing Kpt function gantry-$(functions):$(VERSION))
	docker build -t gantry-$(functions):$(VERSION) .

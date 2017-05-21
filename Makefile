PACKAGES = $(shell glide nv)

.PHONY: test
test:
	go test -race -v $(PACKAGES)

.PHONY: generate
generate:
	go generate $(PACKAGES)
	./scripts/fix-mock-vendor.sh

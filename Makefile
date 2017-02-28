.PHONY: generate
generate:
	go generate $(glide nv)
	./scripts/fix-mock-vendor.sh

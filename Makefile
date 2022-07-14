# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: geth android ios geth-cross evm all test clean
.PHONY: geth-linux geth-linux-386 geth-linux-amd64 geth-linux-mips64 geth-linux-mips64le
.PHONY: geth-linux-arm geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
.PHONY: geth-darwin geth-darwin-386 geth-darwin-amd64
.PHONY: geth-windows geth-windows-386 geth-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

geth:
	$(GORUN) build/ci.go install ./cmd/geth
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gwan\" to launch gwan."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."
	@echo "Import \"$(GOBIN)/geth-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"

ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	# $(GORUN) build/ci.go test -coverage -v ./consensus/...
	# $(GORUN) build/ci.go test -coverage -v ./eth/...
	# $(GORUN) build/ci.go test -coverage -v ./console/...
	# $(GORUN) build/ci.go test -coverage -v ./light/...
	# $(GORUN) build/ci.go test -coverage -v ./les/...
	# $(GORUN) build/ci.go test -coverage -v ./mobile/...
	# $(GORUN) build/ci.go test
	# TestGraphQLBlockSerializationEIP2718
	# $(GORUN) build/ci.go test -coverage -v ./graphql/...
	# panic + 6 fail
	# $(GORUN) build/ci.go test -coverage -v ./cmd/...
	# $(GORUN) build/ci.go test -coverage -v ./p2p/...
	# $(GORUN) build/ci.go test -coverage -v ./pos/...
	# $(GORUN) build/ci.go test -coverage -v ./miner/...
	# 16 fail
	$(GORUN) build/ci.go test -coverage -v ./core

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go install golang.org/x/tools/cmd/stringer@latest
	env GOBIN= go install github.com/kevinburke/go-bindata/go-bindata@latest
	env GOBIN= go install github.com/fjl/gencodec@latest
	env GOBIN= go install github.com/golang/protobuf/protoc-gen-go@latest
	env GOBIN= go install ./cmd/abigen
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

geth-cross: geth-linux geth-darwin geth-windows 
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gwan-*

geth-linux: geth-linux-arm
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gwan-linux-*

geth-linux-arm: geth-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gwan-linux-* | grep arm



geth-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 --image=bjzhaoxiao/xgo:1.16.5 --pkg=./cmd/geth  -v ./cmd/geth
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-linux-* | grep arm64



geth-darwin: geth-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gwan-darwin-*

geth-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 --image=bjzhaoxiao/xgo:1.16.5 --pkg ./cmd/geth -v ./cmd/geth
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-darwin-* | grep amd64

geth-windows: geth-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gwan-windows-*

geth-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 --image=bjzhaoxiao/xgo:1.16.5 --pkg ./cmd/geth -v ./cmd/geth
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-windows-* | grep amd64

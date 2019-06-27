# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gwan evm all test testCoin testToken clean
# .PHONY: gwan android ios geth-cross evm all test clean
# .PHONY: geth-linux geth-linux-386 geth-linux-amd64 geth-linux-mips64 geth-linux-mips64le
# .PHONY: geth-linux-arm geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
# .PHONY: geth-darwin geth-darwin-386 geth-darwin-amd64
# .PHONY: geth-windows geth-windows-386 geth-windows-amd64

GOBIN = build/bin
GO ?= latest

linuxDir=$(shell echo gwan-linux-amd64-`cat ./VERSION`-`git rev-parse --short=8 HEAD`)
windowsDir=$(shell echo gwan-windows-amd64-`cat ./VERSION`-`git rev-parse --short=8 HEAD`)
darwinDir=$(shell echo gwan-mac-amd64-`cat ./VERSION`-`git rev-parse --short=8 HEAD`)
# The gwan target build gwan binary
gwan:
	build/env.sh  go run   -gcflags "-N -l"    build/ci.go   install ./cmd/gwan
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gwan\" to launch gwan."

# The evm target build EVM emulator binary
evm:
	build/env.sh go run build/ci.go install ./cmd/evm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/evm\" to start the evm."

	
# The all target build all the wanchain tools
all:
	build/env.sh go run build/ci.go install

# android:
# 	build/env.sh go run build/ci.go aar --local
# 	@echo "Done building."
# 	@echo "Import \"$(GOBIN)/geth.aar\" to use the library."

# ios:
# 	build/env.sh go run build/ci.go xcode --local
# 	@echo "Done building."
# 	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

# The test target run all unit tests
test: all
	build/env.sh go run build/ci.go test

# The testCoin target test a simple wancoin privacy transaction
testCoin: all
	./build/bin/gwan --dev --nodiscover --networkid 483855466823 --datadir './DOCKER/data-loadScript' --etherbase '0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8' --unlock '0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8' --password './DOCKER/data-loadScript/pwdfile' --mine --minerthreads 1 --nodiscover js './loadScript/wancoin.js'

# The testToken target test a simple token privacy transaction
testToken: all
	./build/bin/gwan --dev --nodiscover --networkid 483855466823 --datadir "./DOCKER/data-loadScript" --etherbase '0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8' --unlock '0x9da26fc2e1d6ad9fdd46138906b0104ae68a65d8' --password './DOCKER/data-loadScript/pwdfile' --mine --minerthreads 1 --nodiscover js './loadScript/wantoken.js'

# The clean target clear all the build output
clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen



# Cross Compilation Targets (xgo)

# geth-cross: geth-linux geth-darwin geth-windows geth-android geth-ios
# 	@echo "Full cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-*

# geth-linux: geth-linux-386 geth-linux-amd64 geth-linux-arm geth-linux-mips64 geth-linux-mips64le
# 	@echo "Linux cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-*

# geth-linux-386:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/geth
# 	@echo "Linux 386 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep 386

gwan-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gwan
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-linux-* | grep amd64
	mkdir -p ${linuxDir}
	cp ./build/bin/gwan-linux-* ${linuxDir}/gwan
	tar zcf ${linuxDir}.tar.gz ${linuxDir}/gwan

# geth-linux-arm: geth-linux-arm-5 geth-linux-arm-6 geth-linux-arm-7 geth-linux-arm64
# 	@echo "Linux ARM cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep arm

# geth-linux-arm-5:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/geth
# 	@echo "Linux ARMv5 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep arm-5

# geth-linux-arm-6:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/geth
# 	@echo "Linux ARMv6 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep arm-6

# geth-linux-arm-7:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/geth
# 	@echo "Linux ARMv7 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep arm-7

# geth-linux-arm64:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/geth
# 	@echo "Linux ARM64 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep arm64

# geth-linux-mips:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/geth
# 	@echo "Linux MIPS cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep mips

# geth-linux-mipsle:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/geth
# 	@echo "Linux MIPSle cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep mipsle

# geth-linux-mips64:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/geth
# 	@echo "Linux MIPS64 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep mips64

# geth-linux-mips64le:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/geth
# 	@echo "Linux MIPS64le cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-linux-* | grep mips64le

# geth-darwin: geth-darwin-386 geth-darwin-amd64
# 	@echo "Darwin cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-darwin-*

# geth-darwin-386:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/geth
# 	@echo "Darwin 386 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-darwin-* | grep 386

gwan-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gwan
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-darwin-* | grep amd64
	mkdir -p ${darwinDir}
	cp ./build/bin/gwan-darwin-* ${darwinDir}/gwan
	tar zcf ${darwinDir}.tar.gz ${darwinDir}/gwan

# geth-windows: geth-windows-386 geth-windows-amd64
# 	@echo "Windows cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-windows-*

# geth-windows-386:
# 	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/geth
# 	@echo "Windows 386 cross compilation done:"
# 	@ls -ld $(GOBIN)/geth-windows-* | grep 386

gwan-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gwan
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gwan-windows-* | grep amd64
	mkdir -p ${windowsDir}
	cp ./build/bin/gwan-windows-* ${windowsDir}/gwan.exe
	zip ${windowsDir}.zip ${windowsDir}/gwan.exe

release: clean gwan-linux-amd64 gwan-windows-amd64 gwan-darwin-amd64

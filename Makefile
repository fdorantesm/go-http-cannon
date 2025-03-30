.PHONY: build

build:
	mkdir -p dist
	# Read the version value from version.txt and pass it to the compiledVersion variable via ldflags
	VERSION=$(shell cat version.txt) && \
	go build -ldflags "-X 'cannon/src/cmd.compiledVersion=$$VERSION'" -o dist/cannon ./src/main.go

install:
	cp dist/cannon ${HOME}/bin/cannon
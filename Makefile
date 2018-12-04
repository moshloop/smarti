build:
	go get -u github.com/golang/dep/cmd/dep
	go get github.com/mitchellh/gox
	dep ensure
	gox -os="darwin linux windows" -arch="amd64"
	upx smarti_*
	mv smarti_darwin_amd64  smarti_osx
	mv smarti_linux_amd64  smarti
	mv smarti_windows_amd64.exe  smarti.exe


install:
	go build -o $(shell basename $(PWD)) main.go
	mv $(shell basename $(PWD)) /usr/local/bin

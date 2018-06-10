build:
	mkdir -p build
	gox -os="darwin linux windows" -arch="amd64"
	mkdir -p build/osx
	mkdir -p build/linux
	mkdir -p build/windows
	mv smarti_darwin_amd64 build/osx/smarti
	mv smarti_amd64 build/linux/smarti
	mv smarti_amd64.exe build/windows/smarti.exe
	cp README.md build/
	zip -r smarti.zip build/*
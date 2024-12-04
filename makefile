all: build

build:
	bin/build_install

debug:
	bin/build_install debug

clean:
	touch _gohome
	chmod +rw _gohome -R
	$(RM) -r _gohome
	$(RM) -r _output

.PHONY: all build clean debug

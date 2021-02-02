# Makefile for building and installing
MAKEFILE_LOCATION := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

all: reportlogger

reportlogger:
	cd reportlogger && make clean && make

install: reportlogger
	install -m0755  $(MAKEFILE_LOCATION)/reportlogger/reportlogger /usr/bin

.PHONY: reportlogger

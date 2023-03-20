.PHONY: clean tidy all daginit

GCCGO             ?= $(shell which gccgo)
GO                ?= $(shell which go)

ifeq ($(GCCGO),)
COMPILER           = gc
COMPILER_FLAGS     = 
LINKER_FLAGS       = -ldflags '-s -w'
else
COMPILER           = gccgo
COMPILER_FLAGS     = -gccgoflags '-Os -s -w'
LINKER_FLAGS       =
endif

all: daginit

daginit:
	go build -compiler $(COMPILER) $(COMPILER_FLAGS) $(LINKER_FLAGS) -o $@ cmd/daginit/main.go

clean:
	rm -fv daginit

tidy:
	go mod tidy
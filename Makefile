.PHONY: clean tidy all daginit

COMPILER           = gc
COMPILER_FLAGS     = 
LINKER_FLAGS       = -ldflags '-s -w'


all: daginit

daginit:
	go build -compiler $(COMPILER) $(COMPILER_FLAGS) $(LINKER_FLAGS) -o $@ cmd/daginit/main.go

clean:
	rm -fv daginit

tidy:
	go mod tidy
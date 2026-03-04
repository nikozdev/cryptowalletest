cmd/%: FORCE
	mkdir -p build/cmd
	GOTMPDIR=build/tmp go build -o build/$@.exe $@/main.go
	./build/$@.exe $(filter-out $@,$(MAKECMDGOALS))

test:
	go test ./...

lint:
	go vet ./...

%:
	@true

FORCE:

.PHONY: FORCE test lint
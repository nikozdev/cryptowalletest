cmd/%: FORCE
	mkdir -p build/cmd
	GOTMPDIR=build/tmp go build -o build/$@.exe $@/main.go
	./build/$@.exe $(filter-out $@,$(MAKECMDGOALS))

test:
	go test ./cmd/... ./internal/... -v -count=1

lint:
	go vet ./cmd/... ./internal/...

%:
	@true

FORCE:

.PHONY: FORCE test lint
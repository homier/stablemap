.DEFAULT: lint
.PHONY: test lint bench


lint:
	golangci-lint run


test:
	go test -race -timeout=5s ./...


benchset:
	go test -bench="Set" -benchmem -count=6 | tee mem.out
	grep -v stableSet mem.out | sed 's,/stdSet,,g' > mem.stdSet.out
	grep -v stdSet mem.out | sed 's,/stableSet,,g' > mem.stableSet.out

	benchstat mem.stdSet.out mem.stableSet.out > benchstat.out


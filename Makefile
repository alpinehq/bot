.PHONY: dev

dev:
	reflex -r "\.go$$" -s -- go run $$(ls *.go | grep -v "_test.go") run

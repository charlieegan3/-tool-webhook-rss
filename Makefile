GO_TEST_ARGS :=
FILE_PATTERN := 'html\|plush\|go\|sql\|Makefile'

.PHONY: test
test:
	go test $(GO_TEST_ARGS) ./...

.PHONY: test_watch
test_watch:
	find . | grep $(FILE_PATTERN) | entr bash -c 'clear; make test'

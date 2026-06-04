.PHONY: deps\:test
deps\:test:
	$(MAKE) -C ../modules build
	$(MAKE) -C ../../host-go build

.PHONY: clean
clean:
	go clean -testcache

.PHONY: test
test:
	@$(MAKE) deps:test
	@$(MAKE) clean
	go test ./...

.PHONY: test\:no-deps
test\:no-deps:
	@$(MAKE) clean
	go test ./...

.PHONY: test\:ci
test\:ci:
# We do not make the deps here, the ci does that seperately to avoid compiling stuff
# multiple times etc.
	gotestsum --format testname ./...

# On-demand, non-gating RSS leak diagnostic (issue #160). Excluded from the suites above
# via the `leakdiag` build tag; run this when investigating a suspected memory leak.
.PHONY: diag\:reset-rss
diag\:reset-rss:
	@$(MAKE) deps:test
	go test -tags leakdiag -run ResetRSSDiagnostic -v -timeout 30m ./node/

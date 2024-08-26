mod:
	go list -m --versions

clean:
	go clean -cache
	go clean -modcache
	go clean -testcache
	go mod tidy
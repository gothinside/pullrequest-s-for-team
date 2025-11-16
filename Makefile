coverage_html:
	go test ./... -coverprofile=coverage.out -coverpkg=./internal/...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test_docker_compose
test_docker_compose: 
	@echo "-- starting docker compose"
	docker compose -f ./test/docker-compose.yml up

mockgen: ### generate mock
	mockery --name=UserRepoInterface --dir=internal/user --output=mocks --outpkg=routermocks
	mockery --name=TeamRepoInterface --dir=internal/team --output=mocks --outpkg=routermocks
	mockery --name=PullRequestRepoInterface --dir=internal/pr --output=mocks --outpkg=routermocks
.PHONY: mockgen

run_test:
	go test ./... -coverprofile=coverage.out -coverpkg=./internal/...
	go tool cover -func=coverage.out
	rm -f coverage.out

.PHONY: docker_compose
docker_compose: 
	@echo "-- starting docker compose"
	docker compose -f docker-compose.yml up
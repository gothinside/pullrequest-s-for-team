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

uint-up:
	docker compose -f ./test/docker-compose.uint.yml up -d
uint-down:
	docker compose -f ./test/docker-compose.uint.yml down -v

run_main_test:
	make uint-up
	sleep 2
	-go test ./test/unit_tests/... -coverprofile=coverage.out -coverpkg=./internal/...
	-go tool cover -func=coverage.out
	-rm -f coverage.out
	sleep 1
	make uint-down

e2e-up:
	docker compose -f ./test/docker-compose.e2e.yml up -d

e2e-down:
	docker compose -f ./test/docker-compose.e2e.yml down -v

e2e-test:
	make e2e-up
	sleep 2
	-go test ./test/ewe -v   
	make e2e-down


.PHONY: docker_compose
docker_compose: 
	@echo "-- starting docker compose"
	docker compose -f docker-compose.yml up


docker_compose_down:
	@echo "-- stopping docker compose"
	docker compose -f docker-compose.yml down
BINARY=engine
test: 
	go test -v -cover -covermode=atomic ./...

engine:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ${BINARY} cmd/api/main.go

unittest:
	go test -short  ./...

test-coverage:
	go test -short -coverprofile cover.out -covermode=atomic ./...
	cat cover.out >> coverage.txt

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

docker:
	docker build -t shortener .

run:
	docker-compose up -d

stop:
	docker-compose down

lint-prepare:
	@echo "Installing golangci-lint" 
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s latest

lint:
	./bin/golangci-lint run \
		--exclude-use-default=false \
		--enable=golint \
		--enable=gocyclo \
		--enable=goconst \
		--enable=unconvert \
		./...

generate-mocks:
	mockgen -source=./internal/url/repository.go -destination=./internal/url/mocks/Repository.go -package=mocks
	mockgen -source=./internal/url/usecase.go -destination=./internal/url/mocks/Usecase.go -package=mocks
	mockgen -source=./internal/user/repository.go -destination=./internal/user/mocks/Repository.go -package=mocks
	mockgen -source=./internal/user/usecase.go -destination=./internal/user/mocks/Usecase.go -package=mocks

authkey:
	go run ./cmd/admin/main.go keygen ./private.pem

migrate:
	go run ./cmd/admin/main.go migrate_mongo

seed: migrate
	go run ./cmd/admin/main.go seed

.PHONY: test engine unittest test-coverage clean docker run stop lint-prepare lint generate-mocks authkey migrate seed
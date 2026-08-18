fmt-check:
	gofmt -l . | tee /tmp/gofmt.out
	test ! -s /tmp/gofmt.out

generate:
	buf generate
	sqlc generate
	wire ./services/commerce/cmd/commerce
	wire ./services/mockpay/cmd/mockpay

generate-check:
	$(MAKE) generate
	git diff --exit-code

unit-test:
	go test ./services/... ./pkg/...

race-test:
	go test -race ./services/... ./pkg/...

integration-test:
	go test -race -tags=integration ./tests/integration/...

build:
	go build ./services/commerce/cmd/commerce
	go build ./services/mockpay/cmd/mockpay

docker-build:
	docker build -f services/commerce/Dockerfile -t commerce:ci .
	docker build -f services/mockpay/Dockerfile -t mockpay:ci .

verify: fmt-check generate-check unit-test race-test build
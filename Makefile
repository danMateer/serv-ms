.PHONY: ensure get post run test

ensure:
	dep ensure

get:
	curl -s -XGET -H "Content-type: application/json" "http://127.0.0.1:8080/metric/foo/sum" | jq .

post:
	curl -s -XPOST -H "Content-type: application/json" "http://127.0.0.1:8080/metric/foo" -d '{"value": 30}' | jq .

run:
	go run main.go

test:
	go test -coverprofile=coverage.out

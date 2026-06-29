.PHONY: build test vet tidy build-desktop web web-dev

web:
	cd web/ui && npm install && npm run build

web-dev:
	cd web/ui && npm install && npm run dev

build: web
	go build -o bin/flowagent ./cmd/flowagent

build-desktop: web
	go build -ldflags "-H windowsgui" -o bin/FlowAgent.exe ./cmd/flowagent-desktop

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

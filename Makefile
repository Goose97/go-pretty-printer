GO_BUILD=go build

build: target/pretty-printer

target/pretty-printer: main.go css_ast.go
	$(GO_BUILD) -o target/pretty-printer main.go css_ast.go

clean:
	rm -rf target/*

rebuild: clean build

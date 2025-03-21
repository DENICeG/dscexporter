PKG := dscexporter
TEST_PKG_LIST := config dscparser exporters scheduler

unittest:
	go test -cover $(addprefix $(PKG)/, $(TEST_PKG_LIST))

run: 
	go run cmd/dscexporter/main.go -c ./config.yaml -d /home/max/code/ds-exporter-test/dsc-data
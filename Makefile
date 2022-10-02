battery_exporter.exe: *.go */*.go
	GOOS=windows go build

.PHONY: clean
clean:
	$(RM) battery_exporter.exe

.PHONY: fmt
fmt: *.go */*.go
	go fmt

.PHONY: install
## WARNING: no systemd in WSL
install: battery_exporter.exe
	install -m 755 battery_exporter.exe /usr/local/bin/
	install -m 644 battery_exporter.service /etc/systemd/system/
	systemctl daemon-reload
	systemctl restart battery_exporter.service
	systemctl enable battery_exporter.service

.PHONY: uninstall
uninstall:
	systemctl disable battery_exporter.service
	systemctl stop battery_exporter.service
	$(RM) /usr/local/bin/battery_exporter.exe
	$(RM) /etc/systemd/system/battery_exporter.service
	systemctl daemon-reload

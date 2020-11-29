app:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o enterpriseportal2
	docker build -f enterpriseportal2.dockerfile -t enterpriseportal2:latest .

database:
	docker build -f postgresql.dockerfile -t enterpriseportal2-db:latest .

clean:
	rm -f enterpriseportal2

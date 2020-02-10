app:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o enterpriseportal2
	docker build -f Dockerfile.enterpriseportal2 -t hilobit:enterpriseportal2 .

database:
	docker build -f Dockerfile.postgresql -t hilobit:enterpriseportal2-db .

clean:
	rm -f enterpriseportal2


all:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o enterpriseportal2
	docker build .

clean:
	rm -f enterpriseportal2


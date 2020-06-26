go:

docker:
	docker build -t httpsproxy2http .
	docker run -it -e HOST=localhost -e ENV=dev -p 80:80 httpsproxy2http

compose:
	docker-compose down
	docker-compose build
	docker-compose up -d

clean:
	docker-compose down

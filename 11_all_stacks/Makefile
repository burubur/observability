docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-reload:
	docker-compose up -d --build microservice-1
	docker-compose up -d --build microservice-2

docker-down:
	docker-compose down --remove-orphans

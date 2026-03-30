up:
	docker compose build coder
	docker compose -f docker-compose.yml up --build

down:
	docker compose -f docker-compose.yml down

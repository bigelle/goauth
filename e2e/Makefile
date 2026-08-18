.PHONY: up down test ci logs

up:
	docker compose up -d --build service

down:
	docker compose down -v

test:
	docker compose build tests
	docker compose run --rm tests

logs:
	docker compose logs -f

# full sequence for CI or a clean run on a new machine
ci: up test down

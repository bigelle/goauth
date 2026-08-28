FROM python:3.12-slim

WORKDIR /app

# Устанавливаем супербыстрый установщик uv из готового образа
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

# 1. Копируем только файлы зависимостей
COPY requirements.txt .

# 2. Устанавливаем зависимости системно через uv (без venv для простоты контейнера)
# Убрали --no-cache-dir: теперь слой на 100% кэшируется в GHA
RUN uv pip install --system -r requirements.txt

# 3. Копируем исходный код тестов (инвалидирует только этот шаг при изменении тестов)
COPY . .

CMD ["pytest", "-v"]

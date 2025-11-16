Для запуска необходимо заполнить .env file(см .env.example)вести make docker_compose. Желательно запускать через docker compose, а не docker-compose.
Чтобы запустить тесты для начала надо поднять тестовое бд. Это можно сделать make test_docker_compose. Т.К бд тестовое .env file не прилагается. Данные можно заполнить вручную.

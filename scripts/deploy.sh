#!/bin/bash

# Скрипт для развертывания на сервере

set -e  # Выход при ошибке

echo "Starting deployment..."

# 1. Pull latest changes
git pull origin main

# 2. Проверяем, установлен ли Docker
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Installing..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    rm get-docker.sh
fi

# 3. Проверяем Docker Compose
if ! command -v docker-compose &> /dev/null; then
    echo "Docker Compose is not installed. Installing..."
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

# 4. Создаем .env файл для продакшена если его нет
if [ ! -f .env.production ]; then
    echo "Creating production .env file..."
    cat > .env.production << EOF
DB_HOST=${DB_HOST_EXTERNAL}
DB_PORT=5432
DB_USER=${PROD_DB_USER}
DB_PASSWORD=${PROD_DB_PASSWORD}
DB_NAME=simple_api_prod
DB_SSL_MODE=require
APP_PORT=8080
APP_ENV=production
JWT_SECRET=${PROD_JWT_SECRET}
EOF
fi

# 5. Загружаем production переменные
if [ -f .env.production ]; then
    export $(cat .env.production | xargs)
fi

# 6. Останавливаем старые контейнеры
docker-compose down || true

# 7. Удаляем старые образы
docker system prune -f

# 8. Собираем и запускаем
docker-compose build --no-cache
docker-compose up -d

# 9. Проверяем здоровье
echo "Checking services health..."
sleep 30

if curl -f http://localhost:8080/health; then
    echo "✅ Application is healthy"
else
    echo "❌ Application health check failed"
    docker-compose logs api
    exit 1
fi

# 10. Настройка firewall (если нужно)
# sudo ufw allow 80/tcp
# sudo ufw allow 443/tcp
# sudo ufw allow 8080/tcp

echo "✅ Deployment completed successfully!"
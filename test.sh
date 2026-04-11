docker compose up -d

echo "sleeping to wait for db to setup"
sleep 5
echo "woke up"

export DSN='host=localhost port=5431 user=postgres password=postgres sslmode=disable'
export CACHE_ADDR='localhost:6379'
export CACHE_PASSWD='redis21q'

go test ./...

docker compose down

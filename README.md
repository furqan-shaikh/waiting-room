# Register redis function
1. From terminal run: `cat <filename>.lua | redis-cli -x FUNCTION LOAD`

# Execute redis function
1. Run: `redis-cli`
2. Invoke the function: `FCALL execute_v2 0 roomid1 3 sessiontoken 5`

# Delete a function
1. Run: `redis-cli`
2. Run: `FUNCTION DELETE <library name>`

# Postgres
# Handling migrations using golang-migrate
- Install golang-migrate: `brew install golang-migrate`
- Run postgres locally using docker and set username, password and database
- migrate create -ext sql -dir dao/migrations -seq create_waitingrooms_table
    - If there were no errors, we should have two files available under db/migrations folder:
        - 000001_create_waitingrooms_table.down.sql
        - 000001_create_waitingrooms_table.up.sql

- Create table in .up.sql
- Delete table in .down.sql
- By adding `IF EXISTS/IF NOT EXISTS` we are making migrations idempotent
- Set environment variable: export POSTGRESQL_URL='postgres://<username>:<password>@localhost:5432/waitingroomdb?sslmode=disable'
- Run migrations: migrate -database ${POSTGRESQL_URL} -path dao/migrations up
- Reference: https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md

# Commands
- export PG_DATABASE_URL="postgres://wr:wr@localhost:5432/waitingroomdb?sslmode=disable"
- Control Plane:
    curl -X POST -H "Content-Type: application/json" -d @waitingroom-post.json http://localhost:3000/waitingRooms
    curl http://localhost:3000/waitingRooms/573bba8c-645c-4237-9930-f6c1698956d6
    curl -X DELETE http://localhost:3000/waitingRooms/f3228cc6-db5f-4a69-b7d7-94e1162c22b7

    curl http://localhost:3333/waitingRooms/1/status

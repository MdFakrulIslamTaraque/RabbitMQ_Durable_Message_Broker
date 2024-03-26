# RabbitMQ_Durable_Message_Broker

## Development environment

### Prerequisites
- Docker
- Docker Compose

Run the following command to start the development environment:
```bash
docker compose up --build
```

To inspect the RabbitMQ management console, navigate to `http://0.0.0.0:15672/` in your browser. The default username and password are `guest`.


### Running the application
To check the application, open postman and send a POST request to `http://localhost:1110/user` with the following JSON body:
```json
{
    "name": "John Doe",
    "age": 25
}
```

To check the message queue, navigate to the RabbitMQ management console immediately and log in with the default credentials. You should see a new message in the queue being processed.
Check the database with proper credentials in the docker-compose file to see if the message was successfully processed.

```database Credentials
      - HOST = localhost
      - Port = 5673
      - MYSQL_USER = message_receiver
      - MYSQL_PASSWORD = 12345678
```

### Stopping the development environment
To stop the development environment, run the following command:
```bash
docker compose down
```
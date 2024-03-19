package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Error("%s: %s", msg, err)
	}
}

type User struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	Name       string `json:"name"`
	Age        int    `json:"age"`
	RetryCount int    `json:"retryCount"`
}

func ConnectSQL() *gorm.DB {
	dsn := "message_receiver:12345678@tcp(message_broker_db:3306)/message?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("connection established")
	}
	// Create and migrate db
	err = db.AutoMigrate(&User{})
	if err != nil {
		log.Error("Failed to migrate db")
		return nil
	}
	return db
}
func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// durable" true means that the queue will survive broker restarts, queue name is "hello"
	q, err := ch.QueueDeclare(
		"task_queue2", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// Fair dispatch, with prefetch count 1 means: This tells RabbitMQ not to give more than one message to a worker at a time.
	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	// auto-ack is false, means that the message will be acknowledged after the worker has completed the task after the given time delay
	//         and true means that the message will be acknowledged immediately after the message is received, not after the task is completed
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  //true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	// 2. Save the message to the database
	db := ConnectSQL()

	var forever chan struct{}
	var maxRetries int = 3
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			// 1. Unmarshal the message
			var user User
			err := json.Unmarshal(d.Body, &user)
			if err != nil {
				log.Printf("Failed to unmarshal")
			}

			err = db.Create(&user).Error
			if err != nil {
				user.RetryCount += 1
				if user.RetryCount > maxRetries {
					log.Printf("Max retries reached")
					err = d.Nack(false, false)
					failOnError(err, "Failed to nack a message")
				} else {
					// worker Queue sender
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					fmt.Println(q)

					body, err := json.Marshal(user)
					if err != nil {
						failOnError(err, "Failed to nack a message")
					}
					err = ch.PublishWithContext(ctx,
						"",     // exchange
						q.Name, // routing key
						false,  // mandatory
						false,
						amqp.Publishing{
							DeliveryMode: amqp.Persistent,
							ContentType:  "text/plain",
							Body:         []byte(body),
						})
					failOnError(err, "Failed to publish a message")
					log.Printf(" [x] Sent %s", body)
				}
				log.Printf("Failed to create category")
			}

			// 3. Acknowledge the message
			err = d.Ack(false) // false means that the message will be acknowledged after the worker has completed the task after the given time delay
			failOnError(err, "Failed to acknowledge a message")

			log.Printf("Done with: %v, %v", user.Name, user.Age)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

}

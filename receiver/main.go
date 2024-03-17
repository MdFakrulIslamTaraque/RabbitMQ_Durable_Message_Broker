package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Error("%s: %s", msg, err)
	}
}

type User struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func ConnectSQL() *gorm.DB {
	dsn := "root:12345678@tcp(127.0.0.1:3306)/message?charset=utf8mb4&parseTime=True&loc=Local"
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
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
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

	var forever chan struct{}

	// ================ worker receiver with sleep time ===============================
	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			// 1. Unmarshal the message
			var user User
			err := json.Unmarshal(d.Body, &user)
			if err != nil {
				log.Printf("Failed to unmarshal")
			}

			// 2. Save the message to the database
			db := ConnectSQL()
			err = db.Create(&user).Error
			if err != nil {
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
	"net/http"
	"time"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type User struct {
	ID         int    `gorm:"primaryKey" json:"id"`
	Name       string `json:"name"`
	Age        int    `json:"age"`
	RetryCount int    `json:"retryCount"`
}

var DB *gorm.DB

func CreateUser(c echo.Context) error {

	//creating a connection to the RabbitMQ server
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// creating a channel after the connection is established successfully
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// declaring a queue(named `task_queue`) to send the message to the receiver
	q, err := ch.QueueDeclare(
		"task_queue2", // name
		true,          // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// worker Queue sender
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	fmt.Println(q)

	//body := bodyFrom(os.Args)
	user := &User{}
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	body, err := json.Marshal(user)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
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

	return c.JSON(http.StatusCreated, "message sent successfully")
}

func GetAllUsers(c echo.Context) error {
	var users []User
	if err := DB.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, users)
}

func main() {
	e := echo.New()
	// Routes
	e.GET("/users", GetAllUsers)
	e.POST("/user", CreateUser)
	// Start server
	e.Logger.Fatal(e.Start(":1110"))
}

package main

import (
	"context"
	"fmt"
	"log"
	"producer/model"

	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
	"goyave.dev/goyave/v4"
)

func main() {

	log.Println("Server started")

	if err := goyave.Start(runRoute); err != nil {
		panic(err)
	}
}

func runRoute(router *goyave.Router) {
	router.Post("/send", func(res *goyave.Response, req *goyave.Request) {
		// Send email
		err := sendMessage(req.Request().Context())
		if err != nil {
			res.Error(err)
			return
		}

		res.String(200, "SENT")
	})
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func sendMessage(ctx context.Context) error {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	p := &model.Person{
		Name:  "John",
		Id:    1,
		Email: "johndoe@me.com",
		Phones: []*model.Person_PhoneNumber{
			{Number: "123456789", Type: model.Person_MOBILE},
		},
	}

	out, err := proto.Marshal(p)
	failOnError(err, "Failed to marshal")

	body := out
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		},
	)
	failOnError(err, "Failed to publish a message")

	return nil
}

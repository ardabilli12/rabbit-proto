package main

import (
	"consumer/model"
	"log"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/streadway/amqp"
	"google.golang.org/protobuf/proto"
	"goyave.dev/goyave/v4"
)

type Initialize struct {
	cache *cache.Cache
	r     func(*goyave.Router)
}

func main() {
	ini := &Initialize{
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}

	ini.r = func(r *goyave.Router) {
		ini.getMsgRoute(r)
	}

	log.Println("rabbitmq consumer started")
	go ini.consumer()

	log.Println("http server started")
	if err := goyave.Start(ini.r); err != nil {
		panic(err)
	}

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(err)
	}
}

func (i *Initialize) consumer() {
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

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			p := &model.Person{}
			if err := proto.Unmarshal(d.Body, p); err != nil {
				log.Fatalf("Failed to decode: %s", err)
			}

			log.Printf("Received a message: %s", p)

			i.cache.Set("rabbit", p, cache.DefaultExpiration)
		}
	}()

	log.Println(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

func (i *Initialize) getMsgRoute(router *goyave.Router) *goyave.Router {
	c := i.cache

	router.Get("/", func(resp *goyave.Response, req *goyave.Request) {
		log.Println("function get called")
		resp.String(200, "Hello World!")
	})

	router.Post("/set-foo", func(resp *goyave.Response, req *goyave.Request) {
		// get
		c.Set("foo", "bar", cache.DefaultExpiration)

		resp.String(200, "created")
	})

	router.Get("/get-msg", func(resp *goyave.Response, req *goyave.Request) {
		// get
		val, found := c.Get("rabbit")
		if found {
			var result model.Person

			personBytes, err := proto.Marshal(val.(proto.Message))
			if err != nil {
				resp.String(500, "error marshalling Person to []byte")
				return
			}

			err = proto.Unmarshal(personBytes, &result) //(*model.Person), &result)
			if err != nil {
				resp.String(500, "error unmarshalling JSON")
				return
			}

			resp.JSON(200, result)
		} else {
			resp.String(404, "not found")
		}
	})

	return router
}

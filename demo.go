package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	assistant "github.com/kodernubie/openai-demo-2/1_assistant"
)

func main() {

	log.Println("Open AI Demo 2 - Assistant")

	app := fiber.New()

	assistant.Init(app)

	app.Static("/", "./web")

	app.Listen(":3000")
}

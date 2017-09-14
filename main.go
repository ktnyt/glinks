package main

import (
	"fmt"
	"log"
	"os"

	"github.com/asdine/storm"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"

	_ "github.com/joho/godotenv/autoload"
)

func createMux() (*storm.DB, *echo.Echo) {
	log.Println("Spinning up G-Links")

	// Setup Storm
	dbPath := os.Getenv("DB_PATH")

	if len(dbPath) == 0 {
		dbPath = "."
	}

	db, err := storm.Open(fmt.Sprintf("%s/glinks.db", dbPath))

	if err != nil {
		log.Fatal("Storm DB could not be opened: ", err)
	}

	// Setup Echo
	e := echo.New()

	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	return db, e
}

func main() {
	defer db.Close()

	for _, v := range mappings {
		defer v.Close()
	}

	log.Println("Welcome to G-Links")

	// Setup target and serve
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")

	target := fmt.Sprintf("%s:%s", host, port)

	e.Logger.Debug(e.Start(target))
}

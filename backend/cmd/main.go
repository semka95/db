package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/labstack/echo"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"bitbucket.org/dbproject_ivt/db/backend/internal/middleware"
	_URLHttpDeliver "bitbucket.org/dbproject_ivt/db/backend/internal/url/delivery/http"
	_URLRepo "bitbucket.org/dbproject_ivt/db/backend/internal/url/repository"
	_URLUcase "bitbucket.org/dbproject_ivt/db/backend/internal/url/usecase"
)

func init() {
	viper.SetConfigFile(`config.json`)
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	if viper.GetBool(`debug`) {
		fmt.Println("Service RUN on DEBUG mode")
	}
}

func main() {
	e := echo.New()
	middL := middleware.InitMiddleware()
	e.Use(middL.CORS)

	timeoutContext := time.Duration(viper.GetInt("context.timeout")) * time.Second
	dbPort := viper.GetString(`mongo.port`)
	dbUser := viper.GetString(`mongo.user`)
	dbPass := viper.GetString(`mongo.pwd`)
	dbName := viper.GetString(`mongo.name`)
	uri := fmt.Sprintf("mongodb://%s:%s@mongodb:%s", dbUser, dbPass, dbPort)
	ctx, _ := context.WithTimeout(context.Background(), timeoutContext)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connection problem: ", err)
		os.Exit(1)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatal("Program exit: ", err)
		}
	}()

	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("Ping error: ", err)
	} else {
		fmt.Println("Png: OK")
	}

	ur := _URLRepo.NewMongoURLRepository(client, dbName)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	_URLHttpDeliver.NewURLHandler(e, uu)

	log.Fatal(e.Start(viper.GetString("server.address")))
}

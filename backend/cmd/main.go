package main

import (
	"context"
	// "database/sql"
	"fmt"
	"log"
	"time"

	// _ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// _articleHttpDeliver "bitbucket.org/dbproject_ivt/db/backend/internal/article/delivery/http"
	// _articleRepo "bitbucket.org/dbproject_ivt/db/backend/internal/article/repository"
	// _articleUcase "bitbucket.org/dbproject_ivt/db/backend/internal/article/usecase"
	// _authorRepo "bitbucket.org/dbproject_ivt/db/backend/internal/author/repository"
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
	// dbHost := viper.GetString(`database.host`)
	// dbPort := viper.GetString(`database.port`)
	// dbUser := viper.GetString(`database.user`)
	// dbPass := viper.GetString(`database.pass`)
	// dbName := viper.GetString(`database.name`)
	// connection := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPass, dbHost, dbPort, dbName)
	// val := url.Values{}
	// val.Add("parseTime", "1")
	// val.Add("loc", "Asia/Jakarta")
	// dsn := fmt.Sprintf("%s?%s", connection, val.Encode())
	// dbConn, err := sql.Open(`mysql`, dsn)
	// if err != nil && viper.GetBool("debug") {
	// 	fmt.Println(err)
	// }
	// err = dbConn.Ping()
	// if err != nil {
	// 	log.Fatal(err)
	// 	os.Exit(1)
	// }

	// defer func() {
	// 	err := dbConn.Close()
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }()

	e := echo.New()
	middL := middleware.InitMiddleware()
	e.Use(middL.CORS)

	timeoutContext := time.Duration(viper.GetInt("context.timeout")) * time.Second
	dbPort := viper.GetString(`mongo.port`)
	dbUser := viper.GetString(`mongo.user`)
	dbPass := viper.GetString(`mongo.pwd`)
	dbName := viper.GetString(`mongo.name`)
	uri := fmt.Sprintf("mongodb://%s:%s@localhost:%s", dbUser, dbPass, dbPort)
	ctx, _ := context.WithTimeout(context.Background(), timeoutContext)
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	ur := _URLRepo.NewMongoURLRepository(client, dbName)
	uu := _URLUcase.NewURLUsecase(ur, timeoutContext)
	_URLHttpDeliver.NewURLHandler(e, uu)

	// authorRepo := _authorRepo.NewMysqlAuthorRepository(dbConn)
	// ar := _articleRepo.NewMysqlArticleRepository(dbConn)

	// au := _articleUcase.NewArticleUsecase(ar, authorRepo, timeoutContext)
	// _articleHttpDeliver.NewArticleHandler(e, au)

	log.Fatal(e.Start(viper.GetString("server.address")))
}

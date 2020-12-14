package nv7haven

import (
	"database/sql"
	"os"

	"github.com/Nv7-Github/firebase"
	database "github.com/Nv7-Github/firebase/db"
	_ "github.com/go-sql-driver/mysql" // mysql
	"github.com/gofiber/fiber/v2"
)

// Nv7Haven is the backend for https://nv7haven.tk
type Nv7Haven struct {
	db  *database.Db
	sql *sql.DB
}

func (c *Nv7Haven) routing(app *fiber.App) {
	app.Get("/hella/:input", c.calcHella)
	app.Get("/bestever_new_suggest/:suggestion", c.newSuggestion)
	app.Get("/bestever_get_suggest", c.getSuggestion)
	app.Get("/bestever_vote/:item", c.vote)
	app.Get("/bestever_get_ldb/:len", c.getLdb)
	app.Get("/bestever_refresh", c.refresh)
	app.Get("/bestever_mod", c.deleteBad)
	app.Get("/getmyip", c.getIP)
	app.Get("/new_note/:name/:password", c.newNote)
	app.Get("/new_note/:name/", c.newNote)
	app.Get("/change_note/:name/:password", c.changeNote)
	app.Get("/get_note/:name/:password", c.getNote)
	app.Get("/has_password/:name", c.hasPassword)
	app.Get("/note_search/:query", c.searchNotes)
}

const (
	dbUser     = "u29_c99qmCcqZ3"
	dbPassword = "j8@tJ1vv5d@^xMixUqUl+NmA"
	dbName     = "s29_nv7haven"
)

// InitNv7Haven initializes the handlers for Nv7Haven
func InitNv7Haven(app *fiber.App) error {
	// Firebase DB
	fireapp, err := firebase.CreateAppWithServiceAccount("https://nv7haven.firebaseio.com", "AIzaSyA8ySJ5bATo7OADU75TMfbtnvKmx_g5rSs", []byte(serviceAccount))
	if err != nil {
		return err
	}
	db := database.CreateDatabase(fireapp)

	// SQL Db
	sql, err := sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+os.Getenv("MYSQL_HOST")+":3306)/"+dbName)
	if err != nil {
		panic(err)
	}
	nv7haven := Nv7Haven{
		db:  db,
		sql: sql,
	}

	err = nv7haven.initBestEver()
	if err != nil {
		return err
	}
	nv7haven.routing(app)
	return nil
}

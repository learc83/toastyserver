package server

import (
	"github.com/learc83/toastyserver/database"
	"log"
	"net/http"
)

func StartServer() {
	//WARNING -- DevMode DELETES DB
	database.OpenDBDevMode()

	routes := getRoutes()
	for key, value := range routes {
		http.HandleFunc(key, handlerWrapper(value))
	}

	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Fatal("ListenAndServer: ", err)
	}
}

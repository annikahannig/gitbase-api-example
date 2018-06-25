package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mhannig/gitbase"
)

func usage() {
	flag.PrintDefaults()
	os.Exit(-1)
}

func main() {
	fmt.Println("Gitbase API Example\t\t\t\t\t\tv.23.042.1")

	// Initialize configuration
	config := parseFlags()
	if config.RepoPath == "" {
		usage()
	}

	// Setup repository
	repo, err := gitbase.NewRepository(config.RepoPath)
	if err != nil {
		log.Fatal("Could not initialize or open repo:", err)
		return
	}

	// Setup API
	router := httprouter.New()
	apiRegisterRoutes(&ApiContext{
		Repository: repo,
	}, router)

	router.GET("/",
		func(res http.ResponseWriter, req *http.Request, _ httprouter.Params) {
			fmt.Fprintf(res, "WELCOME TO A DEMO GITBASE")
		})

	log.Println("Listening for HTTP connections on", config.Http.Listen)
	http.ListenAndServe(config.Http.Listen, router)
}

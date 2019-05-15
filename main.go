package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var flags = struct {
	port     string
	username string
	password string
}{
	port:     "port",
	username: "username",
	password: "password",
}

var messages []*SMS

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// using standard library "flag" package
	flag.Int(flags.port, 8080, "port number")
	flag.String(flags.username, "", "the username for the sms service")
	flag.String(flags.password, "", "the password for the sms service")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	port := viper.GetInt(flags.port)
	username := viper.GetString(flags.username)
	password := viper.GetString(flags.password)

	r := mux.NewRouter()

	r.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		sendJSON(w, messages)
	})

	r.HandleFunc("/Accounts/{username}/Messages.json",
		BasicAuth(func(w http.ResponseWriter, r *http.Request) {
			messages = append(messages, &SMS{
				To:   r.FormValue("To"),
				From: r.FormValue("From"),
				Body: r.FormValue("Body"),
			})
		}, username, password))

	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
	}()

	log.Printf("server started on localhost:%d\n", port)
	<-stop
}

type SMS struct {
	To   string `json:"to"`
	From string `json:"from"`
	Body string `json:"body"`
}

// BasicAuth wraps a handler requiring HTTP basic auth for it
func BasicAuth(handler http.HandlerFunc, username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok {
			if user != username || pass != password { // dont care about timings of the checks
				w.WriteHeader(401)
				w.Write([]byte("Unauthorised.\n"))
				return
			}
		}

		handler(w, r)
	}
}

func sendJSON(w http.ResponseWriter, p interface{}) {
	err := json.NewEncoder(w).Encode(p)
	if err != nil {
		http.Error(w, "marshalling json response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
}

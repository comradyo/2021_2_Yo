package main

import (
	deliveryAuth "backend/auth/delivery/http"
	localStorageAuth "backend/auth/repository/localstorage"
	useCaseAuth "backend/auth/usecase"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

func mainPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	fmt.Fprintln(w, "Главная страница")
	http.Redirect(w,r,"http://bmstusa.herokuapp.com/",http.StatusSeeOther)
}

func main() {
	log.Println("Hello, World!")

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	r := mux.NewRouter()

	repo := localStorageAuth.NewRepositoryUserLocalStorage()
	useCase := useCaseAuth.NewUseCaseAuth(repo)
	handler := deliveryAuth.NewHandlerAuth(useCase)

	r.HandleFunc("/", mainPage)
	r.HandleFunc("/signup", handler.SignUp).Methods("GET","POST")
	r.HandleFunc("/signin", handler.SignIn).Methods("POST")
	//Нужен метод для SignIn с методом GET

	log.Info("Deploying. Port: ", port)
	err := http.ListenAndServe(":"+port, r)
	if err != nil {
		log.Error("main error: ", err)
	}



}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database

const(
	hostname string = "localhost:5000" 
	dbName string = "drsims_todo_db"
	collectionName string = "todo"
	port string = ":8080"
)
           
type(
	todoModel struct{
		ID bson.ObjectId `bson:"_id,omitempty"` 
		Title bson.ObjectId `bson:"title"`
		Completed bson.ObjectId `bson:"successful"`
		DateCreated time.Time `bson:"createdAt"`
	}

	todo struct{
		ID string `json:"id"`
		Title string `json:"title"`
		Completed string `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func init(){
	rnd = renderer.New()
	sess, err := mgo.Dial(hostname)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func homehandler(w http.ResponseWriter, r *http.Request){
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)
}

func fetchTodos(w http.ResponseWriter, r *http.Request){
	todos := []todoModel{}

	if err := db.C(collectionName).Find(bson.M{}).All(&todos); err!=nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to fetch todo",
			"error":err,
		})
		return
	}

	todoList := []todo{}

	for _,t := range todos{
		todoList = append(todoList, todo{
			ID:			t.ID.Hex(),
			Title:		string(t.Title),
			Completed: 	string(t.Completed),
			CreatedAt: 	t.DateCreated,
		})
	}
	rnd.JSON(w, http.StatusAccepted, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request){
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err!=nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}
	
	if t.Title == ""{
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"Title is required",
		})
		return
	}

	tm := todoModel{
		ID: bson.NewObjectId(),
		Title: bson.ObjectId(t.Title),
		Completed: "",
		DateCreated: time.Now(),
	
	}
	if err := db.C(collectionName).Insert(tm); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to save Todo",
			"error": err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message":"Todo created successfully",
		"todo_id": tm.ID.Hex(),
	})
}

func deleteTodo(w http.ResponseWriter, r *http.Request){
		id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id){
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"Invalid id format",
		})
		return	
	}

	if err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id)); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to delete the todo",
			"error": err,
		})
		return	
	}

	rnd.JSON(w, http.StatusAccepted, renderer.M{
		"message":"todo deleted successfully",
	})
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id){
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message" : "Invalid ID format",
		})
		return
	}

	var t todo 

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message":"Title field is required",
		})
		return
	}

	if err := db.C(collectionName).
	Update(
		bson.M{"_id": bson.ObjectIdHex(id)},
		bson.M{"title": t.Title, "completed": t.Completed},
	); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"failed to Update",
			"error": err,
		})
		return
	}
}

func main () {
	stopChan := make(chan os.Signal) //adv codes
	signal.Notify(stopChan, os.Interrupt)

	r := chi.NewRouter()  //these are our routes (r)
	r.Use(middleware.Logger)
	r.Get("/", homehandler)
	r.Mount("/todo", todohandlers())

	srv := &http.Server{   //Define our server
		Addr: 				port,
		Handler: 			r,
		ReadTimeout: 		60 * time.Second,
		WriteTimeout: 		60 * time.Second,
		IdleTimeout : 		60 * time.Second,

	}
	go func() {
		log.Printf("Listening on port %v", port)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen:%s\n", err)
		}
	}()

	//Advanced code will learn it properly 
	
	<- stopChan
	

	log.Println("Shutting down server.......")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
		log.Println("server stopped successfully")
	
}

func todohandlers() http.Handler{
	rg := chi.NewRouter() 
	rg.Group(func(r chi.Router)  {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
		
	})
	return rg
}


func checkErr (err error) {
	if err!=nil{
		log.Fatal(err)
	}
}

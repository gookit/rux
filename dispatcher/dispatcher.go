package dispatcher

import (
	"github.com/gookit/souter"
	"log"
	"net/http"
)

const (
	FavIcon = "/favicon.ico"
)

// Dispatcher
type Dispatcher struct {
	router *souter.Router

	PanicHandler      http.Handler
	NotFoundHandler   http.Handler
	NotAllowedHandler http.Handler
}

// New
func New() *Dispatcher {
	return &Dispatcher{}
}

// NewWithRouter
func NewWithRouter(router *souter.Router) *Dispatcher {
	return &Dispatcher{router: router}
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	next := func(w http.ResponseWriter, r *http.Request) {}
	d.CallMdlStack(w, r, next)
}

func (d *Dispatcher) CallMdlStack(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	next(w, r)
}

func Dispatch(w http.ResponseWriter, r *http.Request) {

}

// Run
// d := dispatcher.New()
// d.Run(":8080")
func (d *Dispatcher) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, d))
}

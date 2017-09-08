package pihen

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
)

// RESTMethod is a function type to call when receiving an HTTP request.
type RESTMethod func(*http.Request, context.Context, *user.User) (interface{}, error)

// RESTCollection is a collection of RESTMethods that are bound to a URL prefix.
type RESTCollection struct {
	// URLPrefix e.g. "/api/mycollection".
	URLPrefix     string
	Methods       map[string]RESTMethod
	AllowedOrigin string
}

// RESTErr represents a request error that should be returned from a RESTMethod. Unexpected
// errors should bubble up unchanged.
type RESTErr struct {
	Status  int
	Message string
}

func (e RESTErr) Error() string {
	return fmt.Sprintf("%s (%d)", e.Message, e.Status)
}

// Bind binds RESTCollections as HTTP handlers.
func Bind(collections []RESTCollection) {
	for _, c := range collections {
		http.Handle(c.URLPrefix, httpHandler{c})
	}
}

type httpHandler struct {
	Collection RESTCollection
}

func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	//u := user.Current(c)
	w.Header().Set("Access-Control-Allow-Origin", h.Collection.AllowedOrigin)
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		methods := make([]string, 0, len(h.Collection.Methods))
		for m := range h.Collection.Methods {
			methods = append(methods, m)
		}
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
		return
	}
	//if u == nil {
	//  writeErr(w, ErrNoLogin, c)
	//  return
	//}
	m, ok := h.Collection.Methods[r.Method]
	if !ok {
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return
	}
	resp, err := m(r, c, nil)
	if err != nil {
		switch err := err.(type) {
		case RESTErr:
			log.Infof(c, "Api failure: %d %s", err.Status, err.Message)
			http.Error(w, err.Message, err.Status)
		default:
			log.Errorf(c, "Unexpected error: %s", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-type", "text/json; charset=utf-8")
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.Encode(resp)
}

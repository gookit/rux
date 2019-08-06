package binder

import (
	"encoding/json"
	"net/http"
)

var JSON = BindFunc(func(i interface{}, r *http.Request) error {
	return json.NewDecoder(r.Body).Decode(i)
})

func BindJSON(i interface{}, r *http.Request) error {
	return JSON.Bind(i, r)
}

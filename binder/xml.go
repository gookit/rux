package binder

import (
	"encoding/xml"
	"net/http"
)

var XML = BindFunc(func(i interface{}, r *http.Request) error {
	return xml.NewDecoder(r.Body).Decode(i)
})

func BindXML(i interface{}, r *http.Request) error {
	return XML.Bind(i, r)
}

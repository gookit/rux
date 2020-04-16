package render

import (
	"encoding/json"

	"github.com/gookit/rux"
)

// JSON response rendering
func JSON() {

}

// JSONP response rendering
func JSONP(status int, callback string, ptr interface{}, c *rux.Context) error {
	enc := json.NewEncoder(c.Resp)

	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(rux.ContentType, "application/javascript; charset=UTF-8")

	var err error
	if _, err = c.Resp.Write([]byte(callback + "(")); err != nil {
		return err
	}

	if err = enc.Encode(ptr); err != nil {
		return err
	}

	_, err = c.Resp.Write([]byte(");"))
	return err
}

package render

import (
	"encoding/json"

	"github.com/gookit/rux"
)

// JSON response rendering
func JSON(status int, ptr interface{}, c *rux.Context) error {
	bs, err := json.Marshal(ptr)
	if err != nil {
		return err
	}

	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(rux.ContentType, "application/json; charset=UTF-8")

	if len(bs) > 0 {
		c.WriteBytes(bs)
	}

	return nil
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

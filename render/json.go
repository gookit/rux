package render

import "encoding/json"

// JSON response rendering
func JSON() {

}

// JSONP response rendering
func JSONP(status int, callback string, i interface{}) {
	enc := json.NewEncoder(c.Resp)

	c.Resp.WriteHeader(status)
	c.Resp.Header().Set(ContentType, "application/javascript; charset=UTF-8")

	var err error
	if _, err = c.Resp.Write([]byte(callback + "(")); err != nil {
		panic(err)
	}

	if err = enc.Encode(i); err != nil {
		panic(err)
	}

	if _, err = c.Resp.Write([]byte(");")); err != nil {
		panic(err)
	}
}

package binding_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gookit/goutil/netutil/httpctype"
	"github.com/gookit/goutil/testutil"
	"github.com/gookit/goutil/testutil/assert"
	"github.com/gookit/rux/binding"
)

var (
	userQuery = "age=12&name=inhere"
	userJSON  = `{"age": 12, "name": "inhere"}`
	userXML   = `<?xml version="1.0" encoding="UTF-8" ?>
<body>
    <age>12</age>
    <name>inhere</name>
</body>`
)

type User struct {
	Age  int    `query:"age" form:"age" xml:"age"`
	Name string `query:"name" form:"name" xml:"name"`
}

func TestAuto(t *testing.T) {
	is := assert.New(t)
	r := http.NewServeMux()

	r.HandleFunc("/AutoBind", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := &User{}
		if ctype := r.Header.Get(httpctype.Key); ctype != "" {
			fmt.Printf(" - auto bind data by content type: %s\n", ctype)
		} else {
			fmt.Println(" - auto bind data from URL query string")
		}

		err := binding.Bind(r, u)
		testBoundedUserIsOK(is, err, u)
	}))

	// post Form body
	w := testutil.MockRequest(r, http.MethodPost, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userQuery),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEPOSTForm,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// post JSON body
	w = testutil.MockRequest(r, http.MethodPost, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userJSON),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEJSON,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// post XML body
	w = testutil.MockRequest(r, http.MethodPost, "/AutoBind", &testutil.MD{
		Body: strings.NewReader(userXML),
		Headers: testutil.M{
			httpctype.Key: httpctype.MIMEXML,
		},
	})
	is.Eq(http.StatusOK, w.Code)

	// URL query string
	w = testutil.MockRequest(r, http.MethodGet, "/AutoBind?"+userQuery, nil)
	is.Eq(http.StatusOK, w.Code)
}

func TestHeaderBinder_Bind(t *testing.T) {
	req, err := http.NewRequest("POST", "/", nil)
	is := assert.New(t)
	is.NoErr(err)

	req.Header.Set("age", "12")
	req.Header.Set("name", "inhere")

	u := &User{}
	err = binding.Header.Bind(req, u)
	testBoundedUserIsOK(is, err, u)

	u = &User{}
	err = binding.Header.BindValues(req.Header, u)
	testBoundedUserIsOK(is, err, u)
}

func testBoundedUserIsOK(is *assert.Assertions, err error, u *User) {
	is.NoErr(err)
	is.NotEmpty(u)
	is.Eq(12, u.Age)
	is.Eq("inhere", u.Name)
}

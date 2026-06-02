package main

import (
	"log"
	"net/http"

	"github.com/gookit/rux/v2"
	"github.com/gookit/rux/v2/pkg/binding"
	"github.com/gookit/validate"
)

type validateAdapter struct{}

func (validateAdapter) Validate(obj any) error {
	v := validate.New(obj)
	if v.Validate() {
		return nil
	}
	return v.Errors.OneError()
}

type userForm struct {
	Name  string `json:"name" validate:"required|minLen:2"`
	Email string `json:"email" validate:"required|email"`
}

func main() {
	binding.Validator = validateAdapter{}

	r := rux.New()
	r.POST("/users", func(c *rux.Context) {
		var form userForm
		if err := c.BindJSON(&form); err != nil {
			c.Text(http.StatusBadRequest, err.Error())
			return
		}

		c.JSON(http.StatusOK, map[string]any{"ok": true, "name": form.Name})
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

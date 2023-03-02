# some idea

- add method route by config

```text
func (grp *MyController) AddRoutes(r *rux.Router) {
    r.QuickMapping(map[string]rux.HandlerFunc {
        "/ GET,POST": grp.Index,
 		// "GET" short as "/home GET"
        " GET": grp.Home,
        "/about GET": grp.About,
    })
}

func (grp *MyController) MappingRoutes(r *rux.Router) {
    return map[string]rux.HandlerFunc {
        "/ GET,POST": grp.Index,
        "GET": grp.Home,
        "/about GET": grp.About,
    }
}

func (*MyController) Index(c *rux.Context) {
   
}

func (*MyController) Home(c *rux.Context) {
   
}

func (*MyController) About(c *rux.Context) {
   
}
```

- need read annotations

```text
// AutoLoad auto register routes by a controller struct
func (r *Router) AutoLoad(prefix string, obj any, middles ...HandlerFunc) {
	cv := reflect.ValueOf(obj)
	if cv.Kind() != reflect.Ptr {
		panic("autoload controller must type ptr")
	}

	cv = cv.Elem()
	ct := cv.Type()
	if ct.Kind() != reflect.Struct {
		panic("autoload controller must type struct")
	}

	var actionsMiddleMap = make(map[string][]HandlerFunc)

	// can custom add middleware for actions
	if m := cv.MethodByName("Uses"); m.IsValid() {
		if uses, ok := m.Interface().(func() map[string][]HandlerFunc); ok {
			actionsMiddleMap = uses()
		}
	}

	resName := strings.ToLower(ct.Elem().Name())

	r.Group(prefix, func() {
		methodNum := cv.NumMethod()
		for i := 0; i < methodNum; i++ {
			mv := cv.Method(i)
			if !mv.IsValid() {
				continue
			}

			action, ok := mv.Interface().(func(*Context))
			if !ok {
				continue
			}

			name := mv.Type().Method(i).Name
			var route *Route

			routeName := resName + "_" + strings.ToLower(name)
			if name == IndexAction || name == StoreAction {
				route = r.AddNamed(routeName, "/", action, methods...)
			} else if name == CreateAction {
				route = r.AddNamed(routeName, "/"+strings.ToLower(name)+"/", action, methods...)
			} else if name == EditAction {
				route = r.AddNamed(routeName, "{id}/"+strings.ToLower(name)+"/", action, methods...)
			} else { // if name == SHOW || name == UPDATE || name == DELETE
				route = r.AddNamed(routeName, "{id}/", action, methods...)
			}

			if actionMiddles, ok := actionsMiddleMap[name]; ok {
				route.Use(actionMiddles...)
			}
		}
	}, middles...)
}
```
package render

import (
	"bytes"
	"html/template"
	"strings"
)

var globalTplFuncMap = template.FuncMap{
	// don't escape content
	"raw": func(s string) string {
		return s
	},
	"trim": strings.TrimSpace,
	"join": strings.Join,
	// upper first char
	"upFirst": func(s string) string {
		if len(s) != 0 {
			f := s[0]
			// is lower
			if f >= 'a' && f <= 'z' {
				return strings.ToUpper(string(f)) + string(s[1:])
			}
		}

		return s
	},
}

var layoutTplFuncMap = template.FuncMap{
	// include other template file
	"include": func(filePath string) (template.HTML, error) {
		var buf bytes.Buffer
		t := template.Must(template.New("include").ParseFiles(filePath))

		if err := t.Execute(&buf, nil); err != nil {
			panic(err)
			// return "", nil
		}

		// Return safe HTML here since we are rendering our own template.
		return template.HTML(buf.String()), nil
	},
}

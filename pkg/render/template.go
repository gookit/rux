package render

import "io"

// TemplateRenderer is the abstract interface a HTML template engine must
// satisfy to be plugged into a Responder. Keeping it minimal means any
// engine (easytpl, std html/template, pongo2, …) can be adapted with a
// tiny wrapper, and rux itself stays template-engine-free.
//
//	r := render.New()
//	r.SetTemplateRenderer(myEngineAdapter)
//	r.HTML(w, 200, "home.tpl", data)
type TemplateRenderer interface {
	// Render writes the named template to w with the given data and
	// optional layout. Engines that don't support layouts may ignore
	// the layout argument.
	Render(w io.Writer, name string, data any, layout ...string) error
}

// TemplateLoader is an optional capability — when the configured
// TemplateRenderer also implements TemplateLoader, callers can ask
// Responder to load templates via LoadGlob / LoadFiles. Engines
// without a load step can omit this interface.
type TemplateLoader interface {
	LoadGlob(pattern string) error
	LoadFiles(files ...string) error
}

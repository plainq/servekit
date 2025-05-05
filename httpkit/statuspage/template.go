package statuspage

import (
	"embed"
	"html/template"
	"io"

	"github.com/heartwilltell/hc"
)

var (
	//go:embed status.html
	assets     embed.FS
	statusPage = template.Must(template.ParseFS(assets, "status.html"))
)

// options holds configuration for rendering the status page.
type renderOptions struct {
	err error
}

// Option defines a function that configures the rendering options.
type Option func(*renderOptions)

// WithError provides an option to set an error for rendering.
func WithError(err error) Option {
	return func(o *renderOptions) { o.err = err }
}

// RenderStatus renders the health status page.
// It accepts functional options to customize rendering behavior.
func RenderStatus(w io.Writer, report *hc.ServiceReport, options ...Option) error {
	renderOpts := renderOptions{}

	for _, option := range options {
		option(&renderOpts)
	}

	data := struct {
		Report *hc.ServiceReport
		Error  error
	}{
		Report: report,
		Error:  renderOpts.err,
	}

	return statusPage.Execute(w, data)
}

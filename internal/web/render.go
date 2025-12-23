package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed templates
var templatesFS embed.FS

// PageData is the data structure passed to all templates.
type PageData struct {
	Title           string        // Page title (appended to site name)
	SidebarItems    []SidebarItem // All sidebar items from registry
	ActivePath      string        // Current URL path for highlighting
	Content         any           // Plugin-specific template data
	IsHTMX          bool          // True if partial update request
	RenderedContent template.HTML // Pre-rendered content for shell wrapper
}

// Renderer handles template rendering with HTMX-aware handling.
type Renderer struct {
	templates *template.Template
	registry  *PluginRegistry
}

// NewRenderer creates a new Renderer with templates from the registry's plugins.
func NewRenderer(registry *PluginRegistry) (*Renderer, error) {
	// Create base template with helper functions
	funcMap := template.FuncMap{
		"isActive": func(currentPath, itemPath string) bool {
			if itemPath == "/" {
				return currentPath == "/"
			}
			return strings.HasPrefix(currentPath, itemPath)
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse shell templates from embedded FS
	shellTemplates, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("getting shell templates: %w", err)
	}

	tmpl, err = tmpl.ParseFS(shellTemplates, "*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing shell templates: %w", err)
	}

	// Parse plugin templates
	for _, p := range registry.All() {
		tp, ok := p.(TemplatePlugin)
		if !ok {
			continue
		}

		pluginFS := tp.Templates()
		pluginID := p.ID()

		// Walk the plugin's templates directory
		err := fs.WalkDir(pluginFS, "templates", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".html") {
				return nil
			}

			// Read template content
			content, err := fs.ReadFile(pluginFS, path)
			if err != nil {
				return fmt.Errorf("reading template %s: %w", path, err)
			}

			// Name template as "pluginID/filename.html"
			baseName := filepath.Base(path)
			templateName := pluginID + "/" + baseName

			_, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return fmt.Errorf("parsing template %s: %w", templateName, err)
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("loading templates for plugin %s: %w", pluginID, err)
		}
	}

	return &Renderer{
		templates: tmpl,
		registry:  registry,
	}, nil
}

// Render renders a template with HTMX-aware handling.
// For HTMX requests: renders content template only.
// For full page: renders shell with content embedded.
func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, tmplName string, data any) error {
	isHTMX := req.Header.Get("HX-Request") == "true"

	pageData := PageData{
		SidebarItems: r.registry.SidebarItems(),
		ActivePath:   req.URL.Path,
		Content:      data,
		IsHTMX:       isHTMX,
	}

	if isHTMX {
		// Render just the content template for partial updates
		return r.renderTemplate(w, tmplName, pageData)
	}

	// For full page load, render content first, then wrap in shell
	var contentBuf bytes.Buffer
	if err := r.renderTemplate(&contentBuf, tmplName, pageData); err != nil {
		return err
	}

	pageData.RenderedContent = template.HTML(contentBuf.String())
	return r.renderTemplate(w, "shell.html", pageData)
}

// RenderPartial always renders just the template (no shell wrapper).
func (r *Renderer) RenderPartial(w http.ResponseWriter, tmplName string, data any) error {
	pageData := PageData{
		SidebarItems: r.registry.SidebarItems(),
		Content:      data,
		IsHTMX:       true,
	}
	return r.renderTemplate(w, tmplName, pageData)
}

// renderTemplate executes a named template.
func (r *Renderer) renderTemplate(w interface{ Write([]byte) (int, error) }, name string, data any) error {
	tmpl := r.templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("template not found: %s", name)
	}
	return tmpl.Execute(w, data)
}

// rendererContextKey is the context key for the Renderer.
type rendererContextKey struct{}

// GetRenderer retrieves the Renderer from the request context.
func GetRenderer(r *http.Request) *Renderer {
	renderer, _ := r.Context().Value(rendererContextKey{}).(*Renderer)
	return renderer
}

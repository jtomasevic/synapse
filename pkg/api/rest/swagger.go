package rest

import (
	_ "embed"
	"net/http"
)

//go:embed swagger.yaml
var swaggerSpec []byte

// swaggerUIHTML renders a Swagger UI page that loads the embedded spec
// from /swagger.yaml. Uses the Swagger UI CDN — no vendor files needed.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Synapse API – Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *::before, *::after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger.yaml",
      dom_id: "#swagger-ui",
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIBundle.SwaggerUIStandalonePreset,
      ],
      layout: "BaseLayout",
      deepLinking: true,
    });
  </script>
</body>
</html>`

// registerSwaggerRoutes adds the Swagger UI and spec endpoints to the mux.
func registerSwaggerRoutes(mux *http.ServeMux) {
	// Serve the raw OpenAPI spec.
	mux.HandleFunc("GET /swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(swaggerSpec)
	})

	// Serve the Swagger UI page.
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})
}

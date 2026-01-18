package main

import (
	"flag"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/dchf12/chat/trace"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/objx"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data any, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var avatars Avatar = UseAuthAvatar

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse()

	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	r := newRoom(avatars)
	r.tracer = trace.New(os.Stdout)
	go r.run()

	authGroup := e.Group("")
	authGroup.Use(AuthMiddleware())
	authGroup.GET("/", renderTemplate("chat.html"))

	e.GET("/login", renderTemplate("login.html"))
	e.GET("/auth/:action/:provider", loginHandler)
	e.GET("/logout", logoutHandler)
	e.POST("/uploader", uploaderHandler)
	e.GET("/upload", renderTemplate("upload.html"))

	e.Static("/avatars", "avatars")
	e.GET("/room", r.WebSocketHandler)

	e.Logger.Info("Start the web server. Port:", *addr)
	e.Logger.Fatal(e.Start(*addr))
}

func renderTemplate(templateName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		data := map[string]any{
			"Host": c.Request().Host,
		}
		if cookie, err := c.Cookie("auth"); err == nil {
			data["UserData"] = objx.MustFromBase64(cookie.Value)
		}
		return c.Render(http.StatusOK, templateName, data)
	}
}

func logoutHandler(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:   "auth",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

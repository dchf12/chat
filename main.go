package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/dchf12/chat/infra/memory"
	"github.com/dchf12/chat/trace"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	// WebAuthn 初期化
	wconfig := &webauthn.Config{
		RPID:          "localhost",
		RPDisplayName: "ChatterBox",
		RPOrigins:     []string{"http://localhost:8080"},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationPreferred,
		},
	}
	wa, err := webauthn.New(wconfig)
	if err != nil {
		log.Fatalf("failed to initialize WebAuthn: %v", err)
	}

	userRepo := memory.NewUserStore()
	sessionRepo := memory.NewSessionStore()
	passkeyHandler := NewPasskeyHandler(wa, userRepo, sessionRepo)

	authGroup := e.Group("")
	authGroup.Use(AuthMiddleware())
	authGroup.GET("/", renderTemplate("chat.html"))
	authGroup.POST("/uploader", uploaderHandler)
	authGroup.GET("/upload", renderTemplate("upload.html"))

	e.GET("/login", renderTemplate("login.html"))
	e.GET("/auth/:action/:provider", loginHandler)
	e.GET("/logout", logoutHandler)

	// Passkey routes
	e.POST("/passkey/register", passkeyHandler.BeginRegistration)
	e.POST("/passkey/register/finish", passkeyHandler.FinishRegistration)
	e.POST("/passkey/login", passkeyHandler.BeginLogin)
	e.POST("/passkey/login/finish", passkeyHandler.FinishLogin)

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
		if userData, err := getAuthUserData(c); err == nil {
			data["UserData"] = userData
		}
		return c.Render(http.StatusOK, templateName, data)
	}
}

func logoutHandler(c echo.Context) error {
	clearAuthCookie(c)
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

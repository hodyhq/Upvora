package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/getfider/fider/app/models/dto"

	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/i18n"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/tpl"

	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/errors"
)

type clientAssets struct {
	CSS []string
	JS  []string
}

type distAsset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type assetsFile struct {
	Entrypoints struct {
		Main struct {
			Assets []distAsset `json:"assets"`
		} `json:"main"`
	} `json:"entrypoints"`
	ChunkGroups map[string]struct {
		Assets []distAsset `json:"assets"`
	} `json:"namedChunkGroups"`
}

// Renderer is the default HTML Render
type Renderer struct {
	templates     map[string]*template.Template
	assets        *clientAssets
	chunkedAssets map[string]*clientAssets
	mutex         sync.RWMutex
	reactRenderer *ReactRenderer
}

// NewRenderer creates a new Renderer
func NewRenderer() *Renderer {
	reactRenderer, err := NewReactRenderer("ssr.js")
	if err != nil {
		panic(errors.Wrap(err, "failed to initialize SSR renderer"))
	}

	return &Renderer{
		templates:     make(map[string]*template.Template),
		mutex:         sync.RWMutex{},
		reactRenderer: reactRenderer,
	}
}

func (r *Renderer) loadAssets() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.assets != nil && env.IsProduction() {
		return nil
	}

	assetsFilePath := "/dist/assets.json"
	if env.IsTest() {
		// Load a fake assets.json for Unit Testing
		assetsFilePath = "/app/pkg/web/testdata/assets.json"
	}

	jsonFile, err := os.Open(env.Path(assetsFilePath))
	if err != nil {
		return errors.Wrap(err, "failed to open file: assets.json")
	}
	defer func() { _ = jsonFile.Close() }()

	jsonBytes, _ := io.ReadAll(jsonFile)
	file := &assetsFile{}
	err = json.Unmarshal([]byte(jsonBytes), file)
	if err != nil {
		return errors.Wrap(err, "failed to parse file: assets.json")
	}

	r.assets = &clientAssets{
		CSS: make([]string, 0),
		JS:  make([]string, 0),
	}

	r.assets = getClientAssets(file.Entrypoints.Main.Assets)
	r.chunkedAssets = make(map[string]*clientAssets)

	for chunkName, chunkGroup := range file.ChunkGroups {
		r.chunkedAssets[chunkName] = getClientAssets(chunkGroup.Assets)
	}

	return nil
}

func getClientAssets(assets []distAsset) *clientAssets {
	clientAssets := &clientAssets{
		CSS: make([]string, 0),
		JS:  make([]string, 0),
	}

	for _, asset := range assets {
		if strings.HasSuffix(asset.Name, ".map") {
			continue
		}

		assetURL := "/assets/" + asset.Name
		if strings.HasSuffix(asset.Name, ".css") {
			clientAssets.CSS = append(clientAssets.CSS, assetURL)
		} else if strings.HasSuffix(asset.Name, ".js") {
			clientAssets.JS = append(clientAssets.JS, assetURL)
		}
	}

	return clientAssets
}

// Render a template based on parameters
func (r *Renderer) Render(w io.Writer, statusCode int, props Props, ctx *Context) {
	var err error

	if r.assets == nil || env.IsDevelopment() {
		if err := r.loadAssets(); err != nil {
			panic(err)
		}
	}

	public := make(Map)
	private := make(Map)
	if props.Data == nil {
		props.Data = make(Map)
	}

	tenant := ctx.Tenant()
	tenantName := "Upvora"
	defaultTheme := "light"
	if tenant != nil {
		tenantName = tenant.Name
		if tenant.DefaultTheme != "" {
			defaultTheme = tenant.DefaultTheme
		}
	}

	title := tenantName
	if props.Title != "" {
		title = fmt.Sprintf("%s · %s", props.Title, tenantName)
	}

	public["title"] = title

	if props.Description != "" {
		description := strings.ReplaceAll(props.Description, "\n", " ")
		public["description"] = fmt.Sprintf("%.150s", description)
	}

	private["assets"] = r.assets
	private["logo"] = LogoURL(ctx)
	private["defaultTheme"] = defaultTheme
	if tenant != nil {
		if css := buildThemeCSS(tenant); css != "" {
			private["themeCSS"] = template.CSS(css)
		}
	}

	locale := i18n.GetLocale(ctx)
	localeDirection := i18n.GetLocaleDirection(ctx)
	localeChunkName := fmt.Sprintf("locale-%s-client-json", locale)

	// webpack replaces "/" and "." with "-", so we do the same here
	pageChunkName := strings.ReplaceAll(strings.ReplaceAll(props.Page, ".", "-"), "/", "-")
	private["preloadAssets"] = []*clientAssets{
		r.chunkedAssets[localeChunkName],
		r.chunkedAssets[pageChunkName],
	}

	if tenant == nil || tenant.LogoBlobKey == "" {
		private["favicon"] = AssetsURL(ctx, "/static/favicon")
	} else {
		private["favicon"] = AssetsURL(ctx, "/static/favicon/%s", tenant.LogoBlobKey)
	}

	private["currentURL"] = ctx.Request.URL.String()
	if canonicalURL := ctx.Value("Canonical-URL"); canonicalURL != nil {
		private["canonicalURL"] = canonicalURL
	}

	oauthProviders := &query.ListActiveOAuthProviders{
		Result: make([]*dto.OAuthProviderOption, 0),
	}
	if !ctx.IsAuthenticated() && statusCode >= 200 && statusCode < 500 {
		err = bus.Dispatch(ctx, oauthProviders)
		if err != nil {
			panic(errors.Wrap(err, "failed to get list of providers"))
		}
	}

	public["page"] = props.Page
	public["contextID"] = ctx.ContextID()
	public["sessionID"] = ctx.SessionID()
	public["tenant"] = tenant
	public["props"] = props.Data
	public["settings"] = &Map{
		"mode":                env.Config.HostMode,
		"locale":              locale,
		"localeDirection":     localeDirection,
		"version":             env.Version(),
		"environment":         env.Config.Environment,
		"googleAnalytics":     env.Config.GoogleAnalytics,
		"domain":              env.MultiTenantDomain(),
		"hasLegal":            env.HasLegal(),
		"isBillingEnabled":    env.IsBillingEnabled(),
		"baseURL":             ctx.BaseURL(),
		"assetsURL":           AssetsURL(ctx, ""),
		"oauth":               oauthProviders.Result,
		"postWithTags":        env.Config.PostCreationWithTagsEnabled,
		"allowAllowedSchemes": env.Config.AllowAllowedSchemes,
	}

	if ctx.IsAuthenticated() {
		u := ctx.User()
		public["user"] = &Map{
			"id":              u.ID,
			"name":            u.Name,
			"email":           u.Email,
			"role":            u.Role,
			"status":          u.Status,
			"avatarType":      u.AvatarType,
			"avatarURL":       u.AvatarURL,
			"avatarBlobKey":   u.AvatarBlobKey,
			"isAdministrator": u.IsAdministrator(),
			"isCollaborator":  u.IsCollaborator(),
			"isTrusted":       u.IsTrusted,
		}
	}

	templateName := "index.html"

	if ctx.Request.IsCrawler() {
		html, err := r.reactRenderer.Render(ctx.Request.URL, public)
		if err != nil {
			log.Errorf(ctx, "Failed to render react page: @{Error}", dto.Props{
				"Error": err.Error(),
			})
		}
		if html != "" {
			templateName = "ssr.html"
			props.Data["html"] = template.HTML(html)
		}
	}

	tmpl := tpl.GetTemplate("/views/base.html", "/views/"+templateName)
	err = tpl.Render(ctx, tmpl, w, Map{
		"public":  public,
		"private": private,
	})
	if err != nil {
		panic(errors.Wrap(err, "failed to execute template %s", templateName))
	}
}

// buildThemeCSS turns the tenant's Theme settings into CSS TOKEN overrides.
// Values are hex-validated at write time. `html body[...]` outranks the
// stylesheet's body[data-theme] token definitions in both themes, so admin
// colors win without touching any selector the design system owns.
func buildThemeCSS(tenant *entity.Tenant) string {
	var b strings.Builder
	writeTokens := func(prefix, hex string) {
		fmt.Fprintf(&b, "%s-base:%s;", prefix, hex)
		fmt.Fprintf(&b, "%s-light:color-mix(in srgb,%s 72%%,#fff);", prefix, hex)
		fmt.Fprintf(&b, "%s-dark:color-mix(in srgb,%s 78%%,#000);", prefix, hex)
	}
	var body strings.Builder
	if tenant.ThemePrimary != "" {
		writeTokens("--colors-primary", tenant.ThemePrimary)
	}
	for key, hex := range tenant.ThemeAccents {
		if hex == "" {
			continue
		}
		fmt.Fprintf(&b, "--accent-%s:%s;", key, hex)
		fmt.Fprintf(&b, "--accent-%s-grad:linear-gradient(135deg,color-mix(in srgb,%s 75%%,#fff),%s 55%%,color-mix(in srgb,%s 80%%,#000));", key, hex, hex, hex)
	}
	tokens := b.String()
	if tokens == "" {
		return ""
	}
	fmt.Fprintf(&body, "html body{%s}html body[data-theme=\"light\"]{%s}html body[data-theme=\"dark\"]{%s}", tokens, tokens, tokens)
	return body.String()
}

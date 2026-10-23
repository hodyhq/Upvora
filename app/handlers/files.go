package handlers

import (
	"net/http"
	"path"
	"strings"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/web"
)

// ViewAttachmentFile serves a PDF, text or Markdown file attached to an idea,
// only to someone who can see that idea (private ideas included). The file is
// never treated as a page: exact type, nosniff, and a sandbox with nothing
// allowed. Markdown goes out as plain text; the portal renders it sanitized.
func ViewAttachmentFile() web.HandlerFunc {
	return func(c *web.Context) error {
		notFound := func() error {
			c.Response.Header().Set("Cache-Control", "no-cache, no-store")
			c.Response.WriteHeader(http.StatusNotFound)
			return nil
		}
		bkey := c.Param("bkey")
		if !strings.HasPrefix(bkey, "files/") || path.Clean(bkey) != bkey {
			return notFound()
		}

		owner := &query.GetPostIDByAttachment{BlobKey: bkey}
		if err := bus.Dispatch(c, owner); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return notFound()
			}
			return c.Failure(err)
		}
		// The idea query applies the viewer's visibility (private ideas).
		idea := &query.GetPostByID{PostID: owner.Result}
		if err := bus.Dispatch(c, idea); err != nil {
			if errors.Cause(err) == app.ErrNotFound {
				return notFound()
			}
			return c.Failure(err)
		}
		if idea.Result.StatusSlug == enum.PostDeleted.Name() {
			return notFound()
		}
		file := &query.GetBlobByKey{Key: bkey}
		if err := bus.Dispatch(c, file); err != nil {
			return notFound()
		}

		contentType := "text/plain; charset=utf-8"
		if file.Result.ContentType == "application/pdf" {
			contentType = "application/pdf"
		}
		name := path.Base(bkey)
		if i := strings.Index(name, "-"); i >= 0 {
			name = name[i+1:]
		}
		name = strings.NewReplacer(`"`, "", `\`, "", "\r", "", "\n", "").Replace(name)

		h := c.Response.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
		h.Set("Content-Disposition", `inline; filename="`+name+`"`)
		h.Set("Cache-Control", "private, no-store")
		return c.Blob(http.StatusOK, contentType, file.Result.Content)
	}
}

package handlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mock"
)

// mockFile serves one attachment of idea 7; canSee says whether the viewer
// may see that idea.
func mockFile(content []byte, contentType string, canSee bool) {
	mockFileWithStatus(content, contentType, canSee, "open")
}

func mockFileWithStatus(content []byte, contentType string, canSee bool, status string) {
	mockBlob(content, contentType)
	bus.AddHandler(func(ctx context.Context, q *query.GetPostIDByAttachment) error {
		if q.BlobKey != "files/abc-plan.md" && q.BlobKey != "files/abc-spec.pdf" {
			return app.ErrNotFound
		}
		q.Result = 7
		return nil
	})
	bus.AddHandler(func(ctx context.Context, q *query.GetPostByID) error {
		if !canSee || q.PostID != 7 {
			return app.ErrNotFound
		}
		q.Result = &entity.Post{ID: 7, StatusSlug: status}
		return nil
	})
}

func getFile(key string) (int, http.Header, string) {
	code, res := mock.NewServer().OnTenant(mock.DemoTenant).AsUser(mock.AryaStark).
		AddParam("bkey", key).Execute(handlers.ViewAttachmentFile())
	return code, res.Header(), res.Body.String()
}

func TestViewAttachmentFile_ServedLockedDown(t *testing.T) {
	RegisterT(t)
	mockFile([]byte("# Plan\n<script>alert(1)</script>"), "text/markdown; charset=utf-8", true)
	code, h, body := getFile("files/abc-plan.md")
	Expect(code).Equals(http.StatusOK)
	Expect(body).ContainsSubstring("# Plan")
	// Markdown is shown as plain text by the browser; the portal renders it sanitized.
	Expect(h.Get("Content-Type")).Equals("text/plain; charset=utf-8")
	Expect(h.Get("X-Content-Type-Options")).Equals("nosniff")
	Expect(h.Get("Content-Security-Policy")).Equals("default-src 'none'; sandbox")
	Expect(h.Get("Content-Disposition")).Equals(`inline; filename="plan.md"`)
	Expect(h.Get("Cache-Control")).Equals("private, no-store")
}

func TestViewAttachmentFile_PDF(t *testing.T) {
	RegisterT(t)
	mockFile([]byte("%PDF-1.7"), "application/pdf", true)
	code, h, _ := getFile("files/abc-spec.pdf")
	Expect(code).Equals(http.StatusOK)
	Expect(h.Get("Content-Type")).Equals("application/pdf")
	Expect(h.Get("Content-Disposition")).Equals(`inline; filename="spec.pdf"`)
}

func TestViewAttachmentFile_OnlyForThoseWhoSeeTheIdea(t *testing.T) {
	RegisterT(t)
	mockFile([]byte("%PDF-1.7"), "application/pdf", false)
	code, _, body := getFile("files/abc-spec.pdf")
	Expect(code).Equals(http.StatusNotFound)
	Expect(body).Equals("")

	mockFile([]byte("%PDF-1.7"), "application/pdf", true)
	for _, key := range []string{"files/unknown.pdf", "attachments/abc-spec.pdf", "files/../attachments/x.png"} {
		code, _, _ = getFile(key)
		Expect(code).Equals(http.StatusNotFound)
	}
}

// Files never come out of the image route (it has no access check).
func TestViewUploadedImage_RefusesFiles(t *testing.T) {
	RegisterT(t)
	mockBlob([]byte("%PDF-1.7"), "application/pdf")
	for _, key := range []string{"files/abc-spec.pdf", "attachments/sneaky.png"} {
		code, _ := mock.NewServer().OnTenant(mock.DemoTenant).AddParam("bkey", key).Execute(handlers.ViewUploadedImage())
		Expect(code).Equals(http.StatusNotFound)
	}
}

// A deleted idea's files go with it.
func TestViewAttachmentFile_DeletedIdea(t *testing.T) {
	RegisterT(t)
	mockFileWithStatus([]byte("%PDF-1.7"), "application/pdf", true, "deleted")
	code, _, _ := getFile("files/abc-spec.pdf")
	Expect(code).Equals(http.StatusNotFound)
}

package validate_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/getfider/fider/app/models/dto"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/validate"
)

func TestDetectFile(t *testing.T) {
	RegisterT(t)
	pdf := buildPDF("", "")
	for _, tc := range []struct {
		content []byte
		name    string
		ctype   string
		ext     string
	}{
		{pdf, "report.pdf", "application/pdf", ".pdf"},
		{pdf, "report.txt", "application/pdf", ".pdf"}, // the content decides
		{[]byte("hello\n"), "notes.txt", "text/plain; charset=utf-8", ".txt"},
		{[]byte("# Title\n"), "README.MD", "text/markdown; charset=utf-8", ".md"},
		{[]byte("# Title\n"), "plan.markdown", "text/markdown; charset=utf-8", ".md"},
	} {
		ctype, ext, ok := validate.DetectFile(tc.content, tc.name)
		Expect(ok).IsTrue()
		Expect(ctype).Equals(tc.ctype)
		Expect(ext).Equals(tc.ext)
	}
	png, _ := os.ReadFile(env.Path("/app/pkg/web/testdata/logo1.png"))
	for _, tc := range []struct {
		content []byte
		name    string
	}{
		{[]byte("<html><script>x</script></html>"), "page.html"},
		{[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>x</script></svg>`), "a.svg"},
		{[]byte("MZ\x90\x00"), "setup.exe"},
		{[]byte("text with a \x00 byte"), "a.txt"},
		{[]byte{0xff, 0xfe, 0xfd}, "bad.txt"}, // not UTF-8
		{[]byte(""), "empty.txt"},
		{png, "image.txt"},
		{[]byte("hello"), "noext"},
	} {
		_, _, ok := validate.DetectFile(tc.content, tc.name)
		Expect(ok).IsFalse()
	}
}

func fileUpload(content []byte, name string) *dto.ImageUpload {
	return &dto.ImageUpload{Upload: &dto.ImageUploadData{FileName: name, Content: content}}
}

func filesOpts() validate.MultiImageUploadOpts {
	return validate.MultiImageUploadOpts{MaxUploads: 3, MaxKilobytes: 5120, AllowFiles: true}
}

func TestMultiImageUpload_FilesAllowed(t *testing.T) {
	RegisterT(t)
	ctx := context.Background()
	pdf := fileUpload(buildPDF("", ""), "spec.pdf")
	md := fileUpload([]byte("# Plan\n"), "plan.md")
	png, _ := os.ReadFile(env.Path("/app/pkg/web/testdata/logo1.png"))
	img := fileUpload(png, "logo.png")
	messages, err := validate.MultiImageUpload(ctx, nil, []*dto.ImageUpload{pdf, md, img}, filesOpts())
	Expect(err).IsNil()
	Expect(messages).HasLen(0)
	Expect(pdf.Upload.ContentType).Equals("application/pdf")
	Expect(md.Upload.ContentType).Equals("text/markdown; charset=utf-8")

	// a text file given as text (what an assistant writes) rather than bytes
	asText := &dto.ImageUpload{Upload: &dto.ImageUploadData{FileName: "notes.txt", Text: "Notes from the call"}}
	messages, _ = validate.MultiImageUpload(ctx, nil, []*dto.ImageUpload{asText}, filesOpts())
	Expect(messages).HasLen(0)
	Expect(string(asText.Upload.Content)).Equals("Notes from the call")
}

func TestMultiImageUpload_FilesRefused(t *testing.T) {
	RegisterT(t)
	ctx := context.Background()
	for name, upload := range map[string]*dto.ImageUpload{
		"pdf with javascript": fileUpload(jsPDF, "evil.pdf"),
		"too large":           fileUpload(bytes.Repeat([]byte("a"), 5120*1024+1), "big.txt"),
		"html":                fileUpload([]byte("<script>alert(1)</script>"), "x.html"),
	} {
		messages, err := validate.MultiImageUpload(ctx, nil, []*dto.ImageUpload{upload}, filesOpts())
		Expect(err).IsNil()
		if len(messages) == 0 {
			t.Errorf("%s: expected a refusal", name)
		}
	}
	// files need the site switch: without it, only images
	opts := filesOpts()
	opts.AllowFiles = false
	messages, _ := validate.MultiImageUpload(ctx, nil, []*dto.ImageUpload{fileUpload(buildPDF("", ""), "spec.pdf")}, opts)
	Expect(len(messages) > 0).IsTrue()
	// at most 3 attachments of any kind
	four := []*dto.ImageUpload{fileUpload([]byte("a"), "1.txt"), fileUpload([]byte("b"), "2.txt"), fileUpload([]byte("c"), "3.txt"), fileUpload([]byte("d"), "4.md")}
	messages, _ = validate.MultiImageUpload(ctx, nil, four, filesOpts())
	Expect(len(messages) > 0).IsTrue()
}

// Removing the same attachment several times frees one slot, not several.
func TestMultiImageUpload_RepeatedRemovalCountsOnce(t *testing.T) {
	RegisterT(t)
	existing := []string{"files/a-1.txt", "files/b-2.txt", "files/c-3.txt"}
	uploads := []*dto.ImageUpload{
		{BlobKey: "files/a-1.txt", Remove: true}, {BlobKey: "files/a-1.txt", Remove: true}, {BlobKey: "files/a-1.txt", Remove: true},
		fileUpload([]byte("x"), "x.txt"), fileUpload([]byte("y"), "y.txt"), fileUpload([]byte("z"), "z.txt"),
	}
	messages, _ := validate.MultiImageUpload(context.Background(), existing, uploads, filesOpts())
	Expect(len(messages) > 0).IsTrue()

	// one real removal and one upload is fine
	messages, _ = validate.MultiImageUpload(context.Background(), existing, []*dto.ImageUpload{
		{BlobKey: "files/a-1.txt", Remove: true}, fileUpload([]byte("x"), "x.txt"),
	}, filesOpts())
	Expect(messages).HasLen(0)
}

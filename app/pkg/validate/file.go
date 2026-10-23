package validate

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/i18n"
	"github.com/getfider/fider/app/pkg/log"
)

// DetectFile reports the content type and canonical extension of an allowed
// non-image attachment, judged by its content: a PDF by its signature, plain
// text and Markdown by name plus valid UTF-8 without NUL bytes. HTML, SVG and
// anything else are not allowed.
func DetectFile(content []byte, fileName string) (contentType, ext string, ok bool) {
	if bytes.HasPrefix(content, []byte("%PDF-")) {
		return "application/pdf", ".pdf", true
	}
	if len(content) == 0 || !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0 {
		return "", "", false
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".txt", ".text":
		return "text/plain; charset=utf-8", ".txt", true
	case ".md", ".markdown":
		return "text/markdown; charset=utf-8", ".md", true
	}
	return "", "", false
}

// FileUpload validates a non-image attachment: an allowed type, the size
// limit, and for a PDF no active content. On success the content type and the
// file name's extension are set from the content, never from the client.
func FileUpload(ctx context.Context, upload *dto.ImageUpload, maxKilobytes int) []string {
	contentType, ext, ok := DetectFile(upload.Upload.Content, upload.Upload.FileName)
	if !ok {
		return []string{i18n.T(ctx, "validation.custom.unsupportedfileformat")}
	}
	if len(upload.Upload.Content) > maxKilobytes*1024 {
		return []string{i18n.T(ctx, "validation.custom.maxfilesize", i18n.Params{"kilobytes": maxKilobytes})}
	}
	if contentType == "application/pdf" {
		clean, err := ScanPDF(ctx, upload.Upload.Content)
		if errors.Is(err, ErrPDFScanBusy) {
			return []string{i18n.T(ctx, "validation.custom.fileprocessingbusy")}
		}
		if err != nil {
			log.Warnf(ctx, "PDF attachment refused: @{Reason}", dto.Props{"Reason": err.Error()})
			return []string{i18n.T(ctx, "validation.custom.unsafepdf")}
		}
		// Store what was scanned, not the original bytes.
		if len(clean) > maxKilobytes*1024 {
			return []string{i18n.T(ctx, "validation.custom.maxfilesize", i18n.Params{"kilobytes": maxKilobytes})}
		}
		upload.Upload.Content = clean
	}
	upload.Upload.ContentType = contentType
	upload.Upload.FileName = strings.TrimSuffix(filepath.Base(upload.Upload.FileName), filepath.Ext(upload.Upload.FileName)) + ext
	return nil
}

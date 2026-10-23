package validate

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/getfider/fider/app/pkg/errors"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// pdfActiveNames are PDF names that make a document do something by itself
// or carry other files: scripts, launching programs, sending form data,
// opening other documents, embedded files, XFA forms and multimedia. Plain
// links (/URI) stay allowed: they only act when the reader clicks them.
var pdfActiveNames = map[string]bool{
	"JavaScript": true, "JS": true, "Launch": true, "SubmitForm": true, "ImportData": true,
	"GoToR": true, "GoToE": true, "EmbeddedFile": true, "EmbeddedFiles": true, "FileAttachment": true,
	"XFA": true, "RichMedia": true, "RichMediaExecute": true, "Movie": true, "Sound": true,
	"Rendition": true, "Screen": true, "3D": true,
	// Additional actions run on events (opening a page, focusing a field):
	// nothing an attachment needs.
	"AA": true,
}

const (
	pdfScanTimeout      = 10 * time.Second
	pdfMaxDecodeBytes   = 8 << 20 // per stream; decompression bombs stop here
	pdfMaxObjectStreams = 32      // each is decoded to be scanned
	pdfMaxWalkDepth     = 64
	pdfConcurrentScans  = 2
)

// ErrPDFScanBusy means other PDFs are being scanned right now.
var ErrPDFScanBusy = errors.New("PDF scanner busy")

var pdfScanSlots = make(chan struct{}, pdfConcurrentScans)

// ScanPDF parses a PDF fully, including objects packed into compressed object
// streams, and refuses it when it is unreadable, encrypted (it could not be
// inspected) or contains active content (see pdfActiveNames). On success it
// returns the document re-written from what was scanned: objects the
// cross-reference table does not reach, old incremental updates and duplicate
// keys are gone, so no reader can find anything that was not inspected.
func ScanPDF(ctx context.Context, content []byte) (clean []byte, err error) {
	if !bytes.HasPrefix(content, []byte("%PDF-")) {
		return nil, errors.New("not a PDF")
	}
	select {
	case pdfScanSlots <- struct{}{}:
		defer func() { <-pdfScanSlots }()
	default:
		return nil, ErrPDFScanBusy
	}
	defer func() {
		if r := recover(); r != nil {
			clean, err = nil, errors.New("unreadable PDF (%v)", r)
		}
	}()
	c, cancel := context.WithTimeout(ctx, pdfScanTimeout)
	defer cancel()

	if err := scanPDF(c, content); err != nil {
		return nil, err
	}
	pdf, err := readPDF(c, content)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := api.WriteContext(c, pdf, &out); err != nil {
		return nil, errors.Wrap(err, "could not re-write the PDF")
	}
	// Check the re-written file too: it is what is stored and served.
	if err := scanPDF(c, out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func readPDF(c context.Context, content []byte) (*model.Context, error) {
	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	conf.Limits.MaxDecodeBytes = pdfMaxDecodeBytes
	pdf, err := api.ReadContext(c, bytes.NewReader(content), conf)
	if err != nil {
		if pdf != nil && pdf.XRefTable != nil && pdf.XRefTable.Encrypt != nil {
			return nil, errors.New("PDF is encrypted")
		}
		return nil, errors.Wrap(err, "unreadable PDF")
	}
	if pdf.XRefTable.Encrypt != nil || pdf.E != nil {
		return nil, errors.New("PDF is encrypted")
	}
	return pdf, nil
}

func scanPDF(c context.Context, content []byte) error {
	pdf, err := readPDF(c, content)
	if err != nil {
		return err
	}
	streams := map[int]bool{}
	for _, entry := range pdf.XRefTable.Table {
		if entry != nil && entry.ObjectStream != nil {
			streams[*entry.ObjectStream] = true
		}
	}
	if len(streams) > pdfMaxObjectStreams {
		return errors.New("PDF has too many object streams (%d)", len(streams))
	}
	for nr, entry := range pdf.XRefTable.Table {
		if err := c.Err(); err != nil {
			return errors.Wrap(err, "PDF scan took too long")
		}
		if entry == nil || entry.Free {
			continue
		}
		// Resolve through the reader: objects packed into object streams are
		// loaded lazily and would otherwise go unseen.
		gen := 0
		if entry.Generation != nil {
			gen = *entry.Generation
		}
		obj, err := pdf.XRefTable.Dereference(*types.NewIndirectRef(nr, gen))
		if err != nil {
			return errors.Wrap(err, "unreadable PDF object %d", nr)
		}
		if name, found := findPDFActiveName(pdf.XRefTable, obj, 0); found {
			return errors.New("PDF contains active content (%s)", name)
		}
	}
	return nil
}

func findPDFActiveName(xref *model.XRefTable, obj types.Object, depth int) (string, bool) {
	if depth > pdfMaxWalkDepth {
		return "nesting too deep", true
	}
	switch o := obj.(type) {
	case types.Name:
		if pdfActiveNames[o.Value()] {
			return o.Value(), true
		}
	case types.Dict:
		if reason, bad := badPDFAction(xref, o); bad {
			return reason, true
		}
		for key, value := range o {
			if pdfActiveNames[key] {
				return key, true
			}
			if name, found := findPDFActiveName(xref, value, depth+1); found {
				return name, true
			}
		}
	case types.Array:
		for _, value := range o {
			if name, found := findPDFActiveName(xref, value, depth+1); found {
				return name, true
			}
		}
	case types.StreamDict:
		return findPDFActiveName(xref, o.Dict, depth+1)
	}
	return "", false
}

// badPDFAction checks the two actions a plain document may still carry: a
// link (only http, https and mailto) and what happens on opening (only going
// to a page, never fetching a URL).
func badPDFAction(xref *model.XRefTable, d types.Dict) (string, bool) {
	// Values may be indirect references: resolve before judging them.
	resolve := func(obj types.Object) types.Object {
		if ref, isRef := obj.(types.IndirectRef); isRef {
			if resolved, err := xref.Dereference(ref); err == nil {
				return resolved
			}
			return nil
		}
		return obj
	}
	if s, ok := resolve(d["S"]).(types.Name); ok && s.Value() == "URI" {
		uri := strings.ToLower(strings.TrimSpace(pdfText(resolve(d["URI"]))))
		if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "mailto:") {
			return "link that is not a web or mail address", true
		}
	}
	if open, ok := d["OpenAction"]; ok {
		if ref, isRef := open.(types.IndirectRef); isRef {
			resolved, err := xref.Dereference(ref)
			if err != nil {
				return "unreadable open action", true
			}
			open = resolved
		}
		switch action := open.(type) {
		case types.Array: // a destination: open at a page
		case types.Dict:
			if s, _ := resolve(action["S"]).(types.Name); s.Value() != "GoTo" {
				return "action on opening", true
			}
		default:
			return "action on opening", true
		}
	}
	return "", false
}

// pdfText reads a PDF string (literal or hex) as text.
func pdfText(obj types.Object) string {
	switch v := obj.(type) {
	case types.StringLiteral:
		if s, err := types.StringLiteralToString(v); err == nil {
			return s
		}
		return v.Value()
	case types.HexLiteral:
		if b, err := v.Bytes(); err == nil {
			return string(b)
		}
	}
	return ""
}

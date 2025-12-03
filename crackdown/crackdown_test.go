package crackdown

import (
    "io"
    "os"
    // "fmt"
    "log"
    "bytes"
    "testing"
    "strings"

    // "time"
    "math/rand/v2"
)

var fileData string
var rdr strings.Reader

func init(){
    data, err := os.ReadFile("../crackdown.crackdown")
    if err != nil {
        log.Fatalf("os.ReadFile error: %q", err)
    }
    fileData = string(data)
}

func doConvertString(in string) string {
    rdr.Reset(in)
    return strings.Trim(string(ConvertString(&rdr)), "\n")
}

func doConvertStringDiscard(in string) {
    rdr.Reset(in)
    s:=ConvertString(&rdr)
    io.Discard.Write(s)
}

var buf bytes.Buffer 
func doConvertFileDiscardBogus() {
    rdr.Reset(fileData + string([]byte{byte(rand.IntN(100))}))
    // rdr.Reset(fileData + time.Now().UTC().Format("2006-01-02 CET"))
    buf.Reset()
    buf.ReadFrom(&rdr)
    // s:=ConvertString(&rdr)
    io.Discard.Write(buf.Bytes())
}

func doConvertFileDiscard() {
    rdr.Reset(fileData)
    s:=ConvertString(&rdr)
    io.Discard.Write(s)
}

func TestWhitespace(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"\r\n\r\npara\r\n\r\n", "<p>para</p>\n"},
        {"\n\npara\r\nmixed\n\n", "<p>para\nmixed</p>\n"},
        {"\r\n\r\n\r\n\r\n", ""},
        {"\n\n\t\n\t\n\n", "\t\t\n"},
    }
    for _, c := range cases {
        rdr.Reset(c.in)
        got := string(ConvertString(&rdr))
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestInlineMarkupBasic(t *testing.T) {

    cases := []struct {
        in, want string
    }{
        {"", ""},
        {"__italics__", "<i>italics</i>"},
        {"**bold**", "<b>bold</b>"},
        {"~~strike~~", "<s>strike</s>"},
        {"`code`", "<code>code</code>"},
        {"__**~~`nested`~~**__", "<i><b><s><code>nested</code></s></b></i>"},
        {"__**~~`auto closed", "<i><b><s><code>auto closed</code></s></b></i>"},
        {"__n**e~~`s`~~t**e__d", "<i>n<b>e<s><code>s</code></s>t</b>e</i>d"},
        {"_*~no change", "_*~no change"},
        {"`__verbatim__`", "<code>__verbatim__</code>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestBlockMarkupBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"a paragraph", "<p>a paragraph</p>"},
        {"--- para", "<p>--- para</p>"},
        {"---", "<hr/>"},
        {"------", "<hr/>"},
        {"---\n---\n---", "<hr/>\n<hr/>\n<hr/>"},
        {"#h1", "<h1>h1</h1>"},
        {"######h6", "<h6>h6</h6>"},
        {"#**h1b**", "<h1><b>h1b</b></h1>"},
        {"* ul", "<ul><li> ul</li></ul>"},
        {"> bq", "<blockquote> bq</blockquote>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestCode(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"```\n__verbatim__```", "<pre><code>__verbatim__</code></pre>"},
        {"```code```", "<pre><code>code</code></pre>"},
        {"```close on EOF", "<pre><code>close on EOF\n\n</code></pre>"},
        {"```ey<1>```", "<pre><code>ey&lt;1&gt;</code></pre>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
    
}

func TestMultiplineParagraphs(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"multi\npara", "<p>multi\npara</p>"},
        {"multi\n--- para", "<p>multi\n--- para</p>"},
        {"multi\n# para", "<p>multi\n# para</p>"},
        {"multi **li\nne** para", "<p>multi <b>li\nne</b> para</p>"},
        {"multi `li\nne` para", "<p>multi <code>line</code> para</p>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestUnorderedLists(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        // top level
        {"* a\n* b", "<ul><li> a</li><li> b</li></ul>"},
        {"* a\n* b\n* c", "<ul><li> a</li><li> b</li><li> c</li></ul>"},
        // two lists
        {"* a\n\n* a", "<ul><li> a</li></ul><ul><li> a</li></ul>"},
        // nested
        {"* a\n\t* b", "<ul><li> a<ul><li> b</li></ul></li></ul>"},
        {"* a\n\t* b\n\t\t* c", "<ul><li> a<ul><li> b<ul><li> c</li></ul></li></ul></li></ul>"},
        {"* a\n\t* b\n* c", "<ul><li> a<ul><li> b</li></ul></li><li> c</li></ul>"},
        {"* a\n\t* b\n\t\t* c\n\t* d","<ul><li> a<ul><li> b<ul><li> c</li></ul></li><li> d</li></ul></li></ul>"},
        {"* a\n\t* b\n\t\t* c\n* d", "<ul><li> a<ul><li> b<ul><li> c</li></ul></li></ul></li><li> d</li></ul>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestBugOrFeature(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        // bug or not?
        // {"multi \n```code\n``` para", "<p>multi <pre><code>code\n</code></pre>\n para</p>"},
        // these could be prevented with escapes
        {"multi\n> para", "<p>multi<blockquote> para</blockquote></p>"},
        {"multi\n* para", "<p>multi<ul><li> para</li></ul></p>"},
    }
    for _, c := range cases {
        got := doConvertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}


// func BenchmarkList(b *testing.B) {
//     for b.Loop() {
//         mu:="* a\n\t* b\n\t\t* c"
//         doConvertStringDiscard(mu)
//     }
// }

// func BenchmarkList2(b *testing.B) {
//     for b.Loop() {
//         mu := "* a\n\t* b\n\t\t* c* a\n\t* b\n\t\t* c* a\n\t* b\n\t\t* c* a\n\t* b\n\t\t* c* a\n\t* b\n\t\t* c* a\n\t* b\n\t\t* c"
//         doConvertStringDiscard(mu)
//     }
// }

// func BenchmarkFileBase(b *testing.B) {
//     for b.Loop() {
//         doConvertFileDiscardBogus()
//     }
// }

func BenchmarkFileReal(b *testing.B) {
    for b.Loop() {
        doConvertFileDiscard()
    }
}
package crackdown

import (
    "io"
    "os"
    // "fmt"
    "log"
    // "bytes"
    "testing"
    "strings"
)

// func TestMain(m *testing.M) {
//     m.Run()
// }

var testrbuf []byte
var testwbuf []byte
var rdr strings.Reader

func init() {
    data, err := os.ReadFile("../crackdown.crackdown")
    if err != nil {
        log.Fatalf("os.ReadFile error: %q", err)
    }
    testrbuf = append([]byte("\n\n"), data...)
    testwbuf = make([]byte, len(testrbuf)*2)
}

func convertString(in string) string {
    rdr.Reset(in)
    return strings.Trim(string(ConvertString(&rdr)), "\r\n")
}

func converBytesDiscard() {
    io.Discard.Write(ConvertBytes(testrbuf, testwbuf))
}

func TestWhitespace(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"", ""},
        {"\r\n\r\n\r\n\r\n", ""},
        {"\n\n\t\n\t\n\n", "\t"},
        {"\n\npara\r\nmixed\n\n", "<p>para\nmixed</p>"},
        {"\r\n\r\npara\r\n\r\n", "<p>para</p>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestInlineBasic(t *testing.T) {

    cases := []struct {
        in, want string
    }{
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
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestHrBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"---", "<hr/>"},
        {"------", "<hr/>"},
        {"---\n\n---\n\n---", "<hr/>\n<hr/>\n<hr/>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestCodeBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"```\n__verbatim__```", "<pre><code>__verbatim__</code></pre>"},
        {"```code```", "<pre><code>code</code></pre>"},
        {"```close on EOF", "<pre><code>close on EOF</code></pre>"},
        {"```ey<1>```", "<pre><code>ey&lt;1&gt;</code></pre>"},
        {"```multi\nline```", "<pre><code>multi\nline</code></pre>"},
        {"```multi\r\nline```", "<pre><code>multi\r\nline</code></pre>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestHeaderBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"#h1", "<h1>h1</h1>"},
        {"##h2", "<h2>h2</h2>"},
        {"###h3", "<h3>h3</h3>"},
        {"####h4", "<h4>h4</h4>"},
        {"#####h5", "<h5>h5</h5>"},
        {"######h6", "<h6>h6</h6>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestQuoteBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"> bq", "<blockquote> bq</blockquote>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestParaBasic(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"a paragraph", "<p>a paragraph</p>"},
        {"--- para", "<p>--- para</p>"},
        {"multi\nline", "<p>multi\nline</p>"},
        {"####### heh", "<p>####### heh</p>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func TestMultiPara(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"multi\npara", "<p>multi\npara</p>"},
        {"multi\n--- para", "<p>multi\n--- para</p>"},
        {"multi\n# para", "<p>multi\n# para</p>"},
        {"multi **li\nne** para", "<p>multi <b>li\nne</b> para</p>"},
        {"multi `li\nne` para", "<p>multi <code>li\nne</code> para</p>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
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
        {"* ul", "<ul><li>ul</li></ul>"},
        {"* a\n* b", "<ul><li>a</li><li>b</li></ul>"},
        {"* a\n* b\n* c", "<ul><li>a</li><li>b</li><li>c</li></ul>"},
        // two lists
        {"* a\n\n* b", "<ul><li>a</li></ul><ul><li>b</li></ul>"},
        // nested
        {"* a1\n\t* b", "<ul><li>a1<ul><li>b</li></ul></li></ul>"},
        {"* a2\n\t* b\n\t\t* c", "<ul><li>a2<ul><li>b<ul><li>c</li></ul></li></ul></li></ul>"},
        {"* a3\n\t* b\n* c", "<ul><li>a3<ul><li>b</li></ul></li><li>c</li></ul>"},
        {"* a4\n\t* b\n\t\t* c\n\t* d","<ul><li>a4<ul><li>b<ul><li>c</li></ul></li><li>d</li></ul></li></ul>"},
        {"* a5\n\t* b\n\t\t* c\n* d", "<ul><li>a5<ul><li>b<ul><li>c</li></ul></li></ul></li><li>d</li></ul>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
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
        {"multi\n> para", "<p>multi\n<blockquote> para</blockquote></p>"},
        {"multi\n* para", "<p>multi\n<ul><li>para</li></ul></p>"},
        // should create a para
        // {"---\n---\n---", "<hr/>\n<hr/>\n<hr/>"},
    }
    for _, c := range cases {
        got := convertString(c.in)
        if got != c.want {
            t.Errorf("convertString(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

func BenchmarkFileReal(b *testing.B) {
    for b.Loop() {
        converBytesDiscard()
    }
}

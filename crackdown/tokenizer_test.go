package crackdown

import (
    "testing"
    "strings"
)

var rdr2 strings.Reader
func tok(in string) string {
    rdr2.Reset(in)
    return string(Tokenize(&rdr2, len(in)))
}

func TestTokenize(t *testing.T) {
    cases := []struct {
        in, want string
    }{
        {"", ""},
        {"\n", ""},
        {"\r\n", ""},
        {"\r\n\r\n", ""},
        {"a", "\n\na\n\n"},
        {"a\n", "\n\na\n\n"},
        {"\na", "\n\na\n\n"},
        {"\r\na", "\n\na\n\n"},
        {"a\r\n", "\n\na\n\n"},
        {"test\n\ntest", "\n\ntest\n\ntest\n\n"},
        {"test\ntest", "\n\ntest\ntest\n\n"},
        {"\r\ntest\n\ntest\r\n", "\n\ntest\n\ntest\n\n"},
        {"test1\r\ntest2", "\n\ntest1\ntest2\n\n"},

    }
    for _, c := range cases {
        got := tok(c.in)
        if got != c.want {
            t.Errorf("tokenize(%q)\nwanted: %q\ngot:    %q", c.in, c.want, got)
        }
    }
}

package crackdown

import (
    "io"
    // "fmt"
    "sync"
    "bytes"
)

var bufPool = sync.Pool{
    New: func() any {
        return new(bytes.Buffer)
    },
}

type tokenizer struct {
    buf []byte
    i int
    eof bool
}

func Tokenize(r io.Reader, l int) []byte {
    buf:= bufPool.Get().(*bytes.Buffer)
    buf.Reset()
    buf.Grow(max(128, l))
    buf.ReadFrom(r)

    tok:= bufPool.Get().(*bytes.Buffer)
    tok.Reset()
    tok.Grow(max(128,l))
    t:=tok.Bytes()

    tokenizer := tokenizer{buf: buf.Bytes()}
    out := tokenizer.tokenize(t[0:cap(t)])

    bufPool.Put(buf)
    bufPool.Put(tok)

    return out
}

func (t *tokenizer) tokenize(tokens []byte) []byte {
    var (
        lineStart bool = true
        idx int = 0
    )
    
    tokens[idx] = '\n'
    idx++
    tokens[idx] = '\n'
    idx++
    for t.read() == '\n' {}
    t.unread()

    for {
        c := t.read()
        if t.eof {
            break
        }

        if c == '\n' {
            tokens[idx] = '\n'
            idx++
            i := 1
            for {
                c := t.read()
                if c != '\n' {
                    t.unread()
                    break
                }
                if i < 2 {
                    tokens[idx] = '\n'
                    idx++
                    i++
                } 
            }
            lineStart = c == byte('\n')
        } else if lineStart && (c == ' ' || c == '\t') {
            t.unread()
            cnt:=0
            for {
                c := t.read()
                if c == byte(' ') {
                    cnt++
                } else if c == byte('\t') {
                    cnt+=4
                } else {
                    t.unread()
                    break
                }
            }
            for range cnt/4 {
                tokens[idx] = '\t'
                idx++
            }
        } else {
            lineStart = false
            tokens[idx] = c
            idx++
        }
    }
    if idx > 4 {
        if tokens[idx-1] != byte('\n') {
            tokens[idx] = '\n'
            idx++
            tokens[idx] = '\n'
            idx++
        } else if tokens[idx-2] != byte('\n') {
            tokens[idx] = '\n'
            idx++
        }
    }
    return tokens[:idx]
}

func (t *tokenizer) read() byte {
    for t.i < len(t.buf) && t.buf[t.i] == '\r' {
        t.i++
    }
    if t.i >= len(t.buf) {
        t.eof = true
        return 0
    }
    r:=t.buf[t.i]
    t.i++
    return r
}

func (t *tokenizer) unread() {
    if t.i - 1 >= 0 {
        t.i--
    }
}

// stolen from bytes.buffer
func growSlice(b []byte, n int) []byte {
    defer func() {
        if recover() != nil {
            panic("token buffer too large")
        }
    }()
    c := len(b) + n
    if c < 2*cap(b) {
        c = 2 * cap(b)
    }
    b2 := append([]byte(nil), make([]byte, c)...)
    i := copy(b2, b)
    return b2[:i]
}

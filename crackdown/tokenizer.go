package crackdown

import (
    "io"
    "fmt"
    "log"
    // "sync"
    "bytes"
)

type tokenizer struct {
    buf []byte
    i int
    eof bool
}

func init() {
    if false {
        fmt.Print("")
        log.Fatal("")
    }
}

// var bufPool = sync.Pool{
//     New: func() any {
//         return new(bytes.Buffer)
//     },
// }

// func Tokenize(r io.Reader, l int) []byte {
//     buf:= bufPool.Get().(*bytes.Buffer)
//     buf.Reset()
//     buf.Grow(max(128, l))
//     buf.ReadFrom(r)

//     tok:= bufPool.Get().(*bytes.Buffer)
//     tok.Reset()
//     tok.Grow(max(128,l*2))
//     t:=tok.Bytes()

//     tokenizer := tokenizer{buf: buf.Bytes()}
//     out := tokenizer.tokenize(t[0:cap(t)])

//     bufPool.Put(buf)
//     bufPool.Put(tok)

//     return out
// }

var tokenReadBuffer *bytes.Buffer = new(bytes.Buffer)
var tokenWriteBuffer *bytes.Buffer = new(bytes.Buffer)

func Tokenize(r io.Reader, l int) []byte {
    tokenReadBuffer.Reset()
    tokenReadBuffer.Grow(max(128, l))
    tokenReadBuffer.ReadFrom(r)
    buf:= tokenReadBuffer.Bytes()

    tokenWriteBuffer.Reset()
    tokenWriteBuffer.Grow(max(128, l))
    wb:= tokenWriteBuffer.Bytes()

    tokenizer := tokenizer{buf: buf}
    out := tokenizer.tokenize(wb[0:cap(wb)])

    return out
}

func (t *tokenizer) tokenize(tokens []byte) []byte {

    t.buf = bytes.Trim(t.buf, "\r\n")
    if len(t.buf) < 1 {
        return tokens[:0]
    }

    // dst := t.buf[:0]
    // src := t.buf[0:]
    // for {
    //     s:=bytes.IndexByte(src, '\r')
    //     if s < 0 {
    //         dst = append(dst, src...)
    //         break
    //     }
    //     // e:=bytes.IndexByte(src, '\r')
    //     dst = append(dst, src[:s]...)
    //     // for i:=0; i < len(src);
    //     src = src[s+1:]
    // }
    // t.buf = dst

    // b := t.buf[:0]
    // for _, x := range t.buf {
    //     if x != '\r' {
    //         b = append(b, x)
    //     }
    // }
    // t.buf = b
    
    tokens[0] = '\n'
    tokens[1] = '\n'

    r:=0
    w:=2
    m:=len(t.buf)
    // for ; r < m && (t.buf[r] == '\n' || t.buf[r] == '\r'); r++ {}

    for r < m {
        if t.buf[r] == '\n' || t.buf[r] == '\r' {
            nls:=0
            for ; r < m; r++ {
                if t.buf[r] == '\n' {
                    nls++
                } else if t.buf[r] == '\r' {

                } else {
                    break
                }
            }
            if nls > 1 {
                tokens[w] = '\n'
                w++
                tokens[w] = '\n'
                w++
            } else if nls > 0 {
                tokens[w] = '\n'
                w++
            }
        } else if t.buf[r] == ' ' || t.buf[r] == '\t' {
            cnt:=0
            for ; r < m; r++ {
                if t.buf[r] == '\t' {
                    cnt += 4
                } else if t.buf[r] == ' ' {
                    cnt++
                } else {
                    break
                }
            }
            cnt /= 4
            for ; cnt > 0; cnt-- {
                tokens[w] = '\t'
                w++
            }
        } else {
            o := bytes.IndexByte(t.buf[r:], '\n')
            if o < 0 || r + o >= m {
                w += copy(tokens[w:], t.buf[r:])
                break
            }
            if o > 1 && t.buf[r+o-1] == '\r' {
                o--
            }
            w += copy(tokens[w:], t.buf[r:r+o])
            r += o
        }
    }

    tokens = tokens[:w]
    tokens = bytes.TrimRight(tokens, "\n")
    tokens = append(tokens, '\n', '\n')
    return tokens
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

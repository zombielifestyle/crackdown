
package crackdown

import (
    "os"
    // "fmt"
    // "io"
    "log"
    "bytes"
    // "bufio"
    // "html"
    "strings"
    // "unicode"
    "unicode/utf8"
)

type tag struct {
    open []byte
    close []byte
}

type tagNesting struct {
    t, level int8
}

type parser struct {
    tokens []byte
    i int
    ln int
}

type renderer struct {
    // b *strings.Builder
    b *bytes.Buffer
    stack []tagNesting
}

const (
    tagP int8  = iota
    tagH1
    tagH2
    tagH3
    tagH4
    tagH5
    tagH6
    tagUl 
    tagLi
    tagS
    tagB
    tagI
    tagHr
    tagBq
    tagCode
    tagPre
)

var tags = [...]tag {
    tagP:    tag{open: []byte("<p>"), close: []byte("</p>")},
    tagH1:   tag{open: []byte("<h1>"), close: []byte("</h1>")},
    tagH2:   tag{open: []byte("<h2>"), close: []byte("</h2>")},
    tagH3:   tag{open: []byte("<h3>"), close: []byte("</h3>")},
    tagH4:   tag{open: []byte("<h4>"), close: []byte("</h4>")},
    tagH5:   tag{open: []byte("<h5>"), close: []byte("</h5>")},
    tagH6:   tag{open: []byte("<h6>"), close: []byte("</h6>")},
    tagUl:   tag{open: []byte("<ul>"), close: []byte("</ul>")},
    tagLi:   tag{open: []byte("<li>"), close: []byte("</li>")},
    tagS:    tag{open: []byte("<s>"), close: []byte("</s>")},
    tagB:    tag{open: []byte("<b>"), close: []byte("</b>")},
    tagI:    tag{open: []byte("<i>"), close: []byte("</i>")},
    tagHr:   tag{open: []byte("<hr/>"), close: []byte("<hr/>")},
    tagBq:   tag{open: []byte("<blockquote>"), close: []byte("</blockquote>")},
    tagCode: tag{open: []byte("<code>"), close: []byte("</code>")},
    tagPre:  tag{open: []byte("<pre>"), close: []byte("</pre>")},
}

var entities = map[int8][]byte {
    '&':  []byte{'&','a','m','p',';'},
    '\'': []byte{'&','#','3','9',';'},
    '<':  []byte{'&','l','t',';'},
    '>':  []byte{'&','g','t',';'},
    '"':  []byte{'&','#','3','4',';'},
}

var syntaxAsciiSet asciiSet
var entityAsciiSet asciiSet

var stack = make([]tagNesting, 0, 254)
// var builder strings.Builder
var builder bytes.Buffer
var ubuf bytes.Buffer

func init() {
    ubuf.Grow(1024*8)
    builder.Grow(1024*8)
    isASCII:=false
    syntaxAsciiSet, isASCII = makeASCIISet("\n*_~-`#>")
    if !isASCII {
        log.Fatal("syntax asciiset failed")
    }
    entityAsciiSet, isASCII = makeASCIISet("&'<>\"")
    if !isASCII {
        log.Fatal("entity asciiset failed")
    }
}

func ConvertString(s *strings.Reader) []byte {
    builder.Reset()
    parser := parser{tokens:Tokenize(s, s.Len())}
    parser.parse(&renderer{&builder, stack[:0]})
    return builder.Bytes()
}

func ConvertFile(f *os.File) []byte {
    builder.Reset()
    fi, err := f.Stat()
    if err != nil {
        log.Fatalf("stat error: %s", err)
    }
    builder.Grow(int(fi.Size() * 2))
    parser := parser{tokens:Tokenize(f, int(fi.Size() * 2))}
    parser.parse(&renderer{&builder, stack[:0]})
    return builder.Bytes()
}

func (r *renderer) writeString(s string) {
    r.b.WriteString(s)
}

func (r *renderer) writeByte(by byte) {
    r.b.WriteByte(by)
}

func (r *renderer) write(by []byte) {
    r.b.Write(by)
}

func (r *renderer) writeEntityEscaped(b []byte) {
    for {
        i:=indexAnyFast(b, entityAsciiSet)
        if i > -1 {
            r.b.Write(b[:i])
            r.b.Write(entities[int8(b[i])])
            b = b[i+1:]
        } else {
            r.b.Write(b)
            return
        }
    }
}

// func (r *renderer) writeEntityEscaped(b []byte) {
//     for _, c := range b {
//         switch c {
//         case '&', '\'',  '<', '>', '"': 
//             r.b.Write(entities[int8(c)])
//         default:
//             r.b.WriteByte(c)
//         }
//     }
// }

func (r *renderer) open(t int8, level int8) {
    r.write(tags[t].open)
    r.stack = append(r.stack, tagNesting{t, level})
}

func (r *renderer) close() {
    if len(r.stack) > 0 {
        r.write(tags[r.stack[len(r.stack)-1].t].close)
        r.stack = r.stack[0:len(r.stack)-1]
    }
}

func (r *renderer) openOrClose(t int8, level int8) {
    if r.hasTag(t) {
        r.close()
    } else {
        r.open(t, level)
    }
}

func (r *renderer) closeAll() {
    for i:=len(r.stack); i > 0; i-- {
        r.write(tags[r.stack[i-1].t].close)
        r.stack = r.stack[0:i-1]
    }
}

func (r *renderer) hasTag(t int8) bool {
    for i:=len(r.stack); i > 0; i-- {
        if r.stack[i-1].t == t {
            return true
        }
    }
    return false
}

func (r *renderer) getTagNestingLevel(t int8) int8 {
    if len(r.stack) > 0 {
        return r.stack[int8(len(r.stack)-1)].level
    }
    return 0
}

func (p *parser) current() byte {
    if p.i > p.ln {
        return 0
    }
    return p.tokens[p.i]

    // if p.i <= p.ln {
    //     return p.tokens[p.i]
    // }
    // return 0
}

func (p *parser) accept(ch byte) bool {
    if p.i <= p.ln && p.tokens[p.i] == ch {
        p.i++
        return true
    }
    return false
}

func (p *parser) peek() byte {
    if p.i + 1 <= p.ln {
        return p.tokens[p.i+1]
    }
    return 0
}

func (p *parser) skip(n int) {
    p.i += n
}

func (p *parser) peekSlice(n int) []byte {
    i := p.i + 1
    if i > len(p.tokens) {
        return p.tokens[:0]
    } else if i+n >= p.ln {
        return p.tokens[i:]
    }
    return p.tokens[i:i+n]
}

func (p *parser) back() {
    p.i--
}

func (p *parser) count(ch byte) int {
    for i:=p.i; i <= p.ln; i++ {
        if p.tokens[i] != ch {
            return i - p.i
        }
    }
    return 0
}

func (p *parser) parse(r *renderer) {
    p.ln = len(p.tokens) - 1
    /*
    todo:
    - optimize entity escaping
    - how to reduce bounds checks?
    - links, references, footnotes
    - ordered lists
    - entities
    - escaping
    - images
    - todo more flexible rulers
    - blockquote nesting
    - header right hand side decorations?
    - not sure how to handle newlines in `code`
    */
    var indentation int8 = 0
    var startOfLine bool = true
    var startOfBlock bool = true

    for p.i < p.ln {

        if p.current() == '\n' {
            p.skip(1)
            startOfLine = true
            if p.current() == '\n' {
                p.skip(1)
                startOfBlock = true
                r.closeAll()
            }

            indentation = 0
            if p.current() == '\t' {
                for p.accept('\t') {
                    indentation++
                }
            }
            // cases that need startOfLine/startOfBlock are expected to peek
            p.back()
        } else {
            startOfLine = false
            startOfBlock = false
        }

        switch {
        case startOfBlock && p.peek() == '-':
            p.skip(1)
            ubuf.Reset()
            for range p.count('-') - 1 {
                ubuf.WriteByte(p.current())
                p.skip(1)
            }
            if p.peek() == '\n' {
                r.write(tags[tagHr].close)
                r.writeByte(byte('\n'))
            } else {
                ubuf.WriteByte(p.current())
                r.open(tagP, indentation)
                r.write(ubuf.Bytes())
            }
            p.skip(1)
        case startOfBlock && p.peek() == '#':
            p.skip(1)
            cnt := p.count('#')
            for range cnt - 1 {
                p.skip(1)
            }
            if cnt >= 1 && cnt <= 6 {
                r.open(int8(cnt), indentation)
            } else {
                r.open(tagP, indentation)
                r.writeByte(p.current())
            }
            p.skip(1)
        case startOfBlock && string(p.peekSlice(3)) == "```":
            p.skip(4)
            if p.current() == '\n' {
                p.skip(1)
            }
            r.write(tags[tagPre].open)
            r.write(tags[tagCode].open)
            o := bytes.Index(p.tokens[p.i:], []byte("```"))
            if o > -1 {
                r.writeEntityEscaped(p.tokens[p.i:p.i+o])
                p.skip(o+3)
            } else {
                r.writeEntityEscaped(p.tokens[p.i:])
                p.i = p.ln
            }
            r.write(tags[tagCode].close)
            r.write(tags[tagPre].close)
            r.writeByte('\n')
            if p.current() == '\n' {
                p.skip(1)
            }
        case startOfBlock && isLetter(p.peek()):
            p.skip(1)
            if !r.hasTag(tagP) {
                r.open(tagP, indentation)
            }


        case startOfLine && p.peek() == '>':
            p.skip(2)
            r.open(tagBq, indentation)
        case startOfLine && string(p.peekSlice(2)) == "* ":
            p.skip(2)

            if !r.hasTag(tagLi) {
                r.open(tagUl, indentation)
                r.open(tagLi, indentation)
            } else {
                level := r.getTagNestingLevel(tagLi)
                switch {
                case indentation > level:
                    r.open(tagUl, indentation)
                    r.open(tagLi, indentation)
                case indentation == level:
                    r.write([]byte("</li><li>"))
                case indentation < level:
                    for r.stack[len(r.stack)-1].level > indentation {
                        r.close()
                    }
                    if r.hasTag(tagLi) {
                        r.write([]byte("</li><li>"))
                    }
                }
            }


        case p.current() == '`':
            p.skip(1)
            r.write(tags[tagCode].open)
            ubuf.Reset()
            for p.current() != '`' {
                if p.current() != '\n' {
                    ubuf.WriteByte(p.current())
                }
                if p.current() == '\n' && (p.peek() == '\n' || p.peek() == 0) {
                    p.skip(1)
                    break
                }
                p.skip(1)
            }
            r.writeEntityEscaped(ubuf.Bytes())
            r.write(tags[tagCode].close)
            p.skip(1)
        case p.current() == '*' && p.peek() == '*':
            r.openOrClose(tagB, indentation)
            p.skip(2)
        case p.current() == '_' && p.peek() == '_':
            r.openOrClose(tagI, indentation)
            p.skip(2)
        case p.current() == '~' && p.peek() == '~':
            p.skip(2)
            r.openOrClose(tagS, indentation)
        default:
            // i:=bytes.IndexAny(p.tokens[p.i:], "\n*_~-`#>")
            i:=indexAnyFast(p.tokens[p.i:], syntaxAsciiSet)
            if i < 2 || p.i + 1 > p.ln {
                r.writeByte(p.current())
                p.skip(1)
            } else {
                r.write(p.tokens[p.i:p.i+i])
                // r.writeByte(p.current())
                p.skip(i)
            }
        }
    }
    r.closeAll()
}

func isLetter(c byte) bool {
    if (c >= 65 && c <= 90) || (c >= 97 && c <= 122) {
        return true
    }
    return false
    // r, _ := utf8.DecodeRuneInString(s)
    // return unicode.IsLetter(r)
}

// stolen from bytes.IndexAny to use an persistent asciiSet
func indexAnyFast(s []byte, as asciiSet) int {
    for i, c := range s {
        if as.contains(c) {
            return i
        }
    }
    return -1
}

type asciiSet [8]uint32

func (as *asciiSet) contains(c byte) bool {
    return (as[c/32] & (1 << (c % 32))) != 0
}

func makeASCIISet(chars string) (as asciiSet, ok bool) {
    for i := 0; i < len(chars); i++ {
        c := chars[i]
        if c >= utf8.RuneSelf {
            return as, false
        }
        as[c/32] |= 1 << (c % 32)
    }
    return as, true
}

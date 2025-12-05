
package crackdown

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
    "unicode/utf8"

)

type tag struct {
    open []byte
    close []byte
}

type tagNesting struct {
    t, level uint8
}

type parser struct {
    tokens []byte
    i int
    ln int
}

type renderer struct {
    buf []byte
    stack []tagNesting
    w int
}

const (
    tagP uint8 = iota
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

var entitiesEnc = [255][6]uint8{
    '&':  {6, '&','a','m','p',';'},
    '\'': {6, '&','#','3','9',';'},
    '<':  {5, '&','l','t',';'},
    '>':  {5, '&','g','t',';'},
    '"':  {6, '&','#','3','4',';'},
}

var entities = [255]uint8{
    '&':  1,
    '\'': 1,
    '<':  1,
    '>':  1,
    '"':  1,
}

var syntax = [255]uint8{
    '\r': 1,
    '\n': 1,
    '*':  1,
    '_':  1,
    '~':  1,
    '-':  1,
    '`':  1,
    '#':  1,
    '>':  1,
}

var stack = make([]tagNesting, 0, 128)
var writeBuf bytes.Buffer
var readBuf bytes.Buffer

func init() {
    if false {
        fmt.Print("")
    }
    writeBuf.Grow(1024*3)
}

func ConvertString(s *strings.Reader) []byte {
    readBuf.Reset()
    readBuf.Grow(max(128, s.Len()))
    readBuf.WriteByte('\n')
    readBuf.WriteByte('\n')
    readBuf.ReadFrom(s)
    buf:= readBuf.Bytes()

    writeBuf.Reset()
    writeBuf.Grow(s.Len()*2)
    rwb:=writeBuf.Bytes()
    rwb=rwb[0:cap(rwb)]

    parser := parser{tokens:buf}
    r:=&renderer{rwb, stack[:0], 0}
    parser.parse(r)
    return rwb[:r.w]
}

func ConvertFile(f *os.File) []byte {
    fi, err := f.Stat()
    if err != nil {
        log.Fatalf("stat error: %s", err)
    }

    readBuf.Reset()
    readBuf.Grow(max(128, int(fi.Size() * 2)))
    readBuf.WriteByte('\n')
    readBuf.WriteByte('\n')
    readBuf.ReadFrom(f)
    buf:= readBuf.Bytes()

    writeBuf.Reset()
    writeBuf.Grow(int(fi.Size() * 2))
    rwb:=writeBuf.Bytes()
    rwb=rwb[0:cap(rwb)]
    parser := parser{tokens:buf}
    r:=&renderer{rwb, stack[:0], 0}
    parser.parse(r)
    return rwb[:r.w]
}

func (r *renderer) writeByte(by byte) {
    r.buf[r.w] = by
    r.w++
}

func (r *renderer) write(by []byte) {
    r.w += copy(r.buf[r.w:], by)
}

func (r *renderer) writeEntityEscaped(s []byte) {
    for _, c := range s {
        if entities[c] == 1 {
            r.write(entitiesEnc[c][1:entitiesEnc[c][0]])
        } else {
            r.writeByte(c)
        }
    }
}

func (r *renderer) open(t uint8, level uint8) {
    r.write(tags[t].open)
    r.stack = append(r.stack, tagNesting{t, level})
}

func (r *renderer) close() {
    if len(r.stack) > 0 {
        r.write(tags[r.stack[len(r.stack)-1].t].close)
        r.stack = r.stack[0:len(r.stack)-1]
    }
}

func (r *renderer) openOrClose(t uint8, level uint8) {
    if r.hasTag(t) {
        r.close()
    } else {
        r.open(t, level)
    }
}

func (r *renderer) closeAll() {
    for i:=len(r.stack)-1; i >= 0; i-- {
        r.close()
    }
}

func (r *renderer) hasTag(t uint8) bool {
    for i:=len(r.stack); i > 0; i-- {
        if r.stack[i-1].t == t {
            return true
        }
    }
    return false
}

func (r *renderer) getTagNestingLevel(t uint8) uint8 {
    if len(r.stack) > 0 {
        return r.stack[uint8(len(r.stack)-1)].level
    }
    return 0
}

func (p *parser) current() byte {
    if p.i >= p.ln {
        return 0
    }
    return p.tokens[p.i]
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

func (p *parser) back() {
    p.i--
}

func (p *parser) count(ch byte) int {
    i:=p.i
    for ; i < p.ln; i++ {
        if p.tokens[i] != ch {
            break
        }
    }
    return i - p.i
}

func (p *parser) indexSyntax() int {
    for i, c := range p.tokens[p.i:] {
        if syntax[c] == 1 {
            return i
        }
    }
    return -1
}

// func (p *parser) indexSyntax() int {
//     for i:=p.i; i < p.ln; i++ {
//         if syntax[p.tokens[i]] == 1 {
//             return i - p.i
//         }
//     }
//     return -1
// }

func (p *parser) eol() bool {
    if p.i >= p.ln {
        return true
    }
    c:=p.tokens[p.i]
    if c == '\n' || c == '\r' {
        return true
    }
    return false
}

func (p *parser) parse(r *renderer) {
    p.ln = len(p.tokens)

    if bytes.Trim(p.tokens, "\r\n\t ") == nil {
        p.tokens = p.tokens[:0]
        return
    }
    /*
    todo:
    - links, references, footnotes
    - ordered lists
    - entities
    - escaping
    - images
    - todo more flexible rulers
    - blockquote nesting
    - header right hand side decorations?
    */
    var indentation uint8 = 0
    var startOfLine bool = true
    var startOfBlock bool = true
    for p.i < p.ln {

        i:=p.indexSyntax()
        if i < 0 || p.i + i >= p.ln {
            r.write(p.tokens[p.i:])
            p.skip(len(p.tokens[p.i:]))
            break
        } else {
            r.write(p.tokens[p.i:p.i+i])
            p.skip(i)
        }

        startOfBlock = false
        startOfLine  = false
        if p.tokens[p.i] == '\n' || p.tokens[p.i] == '\r' {
            nls:=0
            for ; p.i < p.ln; p.i++ {
                if p.tokens[p.i] == '\n' {
                    nls++
                    continue
                } else if p.tokens[p.i] != '\r' {
                    break
                }
            }
            startOfLine = true
            if nls > 1 {
                startOfBlock = true
                r.closeAll()
            }
            if p.i >= p.ln {
                break
            }
            indentation = 0
            if p.tokens[p.i] == '\t' || p.tokens[p.i] == ' ' {
                cnt:=0
                for ; p.i < p.ln; p.i++ {
                    if p.tokens[p.i] == '\t' {
                        cnt += 4
                        continue
                    } else if p.tokens[p.i] == ' ' {
                        cnt++
                        continue
                    }
                    break
                }
                indentation = uint8(cnt/4)
            }
        }

        if startOfLine && r.hasTag(tagP) {
            r.writeByte('\n')
        }

        switch {
        case startOfBlock && p.current() == '#':
            cnt := p.count('#')
            p.skip(cnt)
            if cnt >= 1 && cnt <= 6 {
                r.open(uint8(cnt), indentation)
                continue
            }
            r.open(tagP, indentation)
            r.write(p.tokens[p.i-cnt:p.i])
        case startOfBlock && p.current() == '-':
            i:=p.count('-')
            p.skip(i)
            if i > 2 && p.eol() {
                r.write(tags[tagHr].close)
                r.writeByte('\n')
                continue
            } 
            r.open(tagP, indentation)
            r.write(p.tokens[p.i-i:p.i])
        case startOfBlock && string(p.tokens[p.i:p.i+3]) == "```":
            p.skip(3)
            if p.current() == '\n' {
                p.skip(1)
            }
            r.write(tags[tagPre].open)
            r.write(tags[tagCode].open)
            i := bytes.Index(p.tokens[p.i:], []byte("```"))
            if i < 0 {
                i = p.ln - p.i
            }
            r.writeEntityEscaped(p.tokens[p.i:p.i+i])
            p.skip(i+3)
            r.write(tags[tagCode].close)
            r.write(tags[tagPre].close)
        case startOfBlock && isLetter(p.current()):
            if !r.hasTag(tagP) {
                r.open(tagP, indentation)
            }
            r.writeByte(p.current())
            p.skip(1)


        case startOfLine && p.tokens[p.i] == '*' && p.peek() == ' ':
            p.skip(2)
            if !r.hasTag(tagLi) {
                r.open(tagUl, indentation)
                r.open(tagLi, indentation)
                continue
            }
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
        case startOfLine && p.current() == '>':
            p.skip(1)
            r.open(tagBq, indentation)


        case p.current() == '*' && p.peek() == '*':
            r.openOrClose(tagB, indentation)
            p.skip(2)
        case p.current() == '_' && p.peek() == '_':
            r.openOrClose(tagI, indentation)
            p.skip(2)
        case p.current() == '~' && p.peek() == '~':
            p.skip(2)
            r.openOrClose(tagS, indentation)
        case p.current() == '`':
            p.skip(1)
            r.write(tags[tagCode].open)
            i := bytes.Index(p.tokens[p.i:], []byte("`"))
            if i < 0 {
                i = len(p.tokens[p.i:])
            }
            r.writeEntityEscaped(p.tokens[p.i:p.i+i])
            r.write(tags[tagCode].close)
            p.skip(i+1)

        default:
            r.writeByte(p.current())
            p.skip(1)
        }

    }
    r.closeAll()
}

func isLetter(c byte) bool {
    if (c >= 65 && c <= 90) || (c >= 97 && c <= 122) {
        return true
    }
    return false
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

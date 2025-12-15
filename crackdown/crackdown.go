
package crackdown

import (
    "os"
    "fmt"
    "log"
    "bytes"
    "strings"
)

type tag struct {
    open []byte
    close []byte
}

type tagNesting struct {
    t, level uint8
}

type stack [256]tagNesting

type parser struct {
    rbuf []byte
    wbuf []byte
    ri int
    rlen int
    wi uint
    stack *stack
    si uint8
    indentation uint8
}

const (
    _ uint8 = iota
    tagH1
    tagH2
    tagH3
    tagH4
    tagH5
    tagH6
    tagP
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
    tagH1:   tag{open: []byte("<h1>"), close: []byte("</h1>")},
    tagH2:   tag{open: []byte("<h2>"), close: []byte("</h2>")},
    tagH3:   tag{open: []byte("<h3>"), close: []byte("</h3>")},
    tagH4:   tag{open: []byte("<h4>"), close: []byte("</h4>")},
    tagH5:   tag{open: []byte("<h5>"), close: []byte("</h5>")},
    tagH6:   tag{open: []byte("<h6>"), close: []byte("</h6>")},
    tagP:    tag{open: []byte("<p>"), close: []byte("</p>")},
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

var entitiesEnc = [256][]uint8{
    '&':  {'&','a','m','p',';'},
    '\'': {'&','#','3','9',';'},
    '<':  {'&','l','t',';'},
    '>':  {'&','g','t',';'},
    '"':  {'&','#','3','4',';'},
}

var entities = [256]uint8{
    '&':  1,
    '\'': 1,
    '<':  1,
    '>':  1,
    '"':  1,
}

var syntax = [256]uint8{
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

var writeBuf bytes.Buffer
var readBuf bytes.Buffer
var sstack *stack = &stack{}

func init() {
    if false {
        fmt.Print("")
    }
    writeBuf.Grow(1024*3)
}

func ConvertBytes(rbuf []byte, wbuf []byte) []byte {
    p := &parser{rbuf,wbuf[0:cap(wbuf)],0,0,0,&stack{},0,0}
    p.parse()
    return wbuf[:p.wi]
}

func ConvertString(s *strings.Reader) []byte {
    readBuf.Reset()
    readBuf.Grow(max(128, s.Len()))
    readBuf.WriteByte('\n')
    readBuf.WriteByte('\n')
    readBuf.ReadFrom(s)
    rbuf := readBuf.Bytes()

    writeBuf.Reset()
    writeBuf.Grow(s.Len()*2)
    wbuf := writeBuf.Bytes()

    p := &parser{rbuf,wbuf[0:cap(wbuf)],0,0,0,&stack{},0,0}
    p.parse()
    return wbuf[:p.wi]
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
    rbuf := readBuf.Bytes()

    writeBuf.Reset()
    writeBuf.Grow(int(fi.Size() * 2))
    wbuf := writeBuf.Bytes()

    p := &parser{rbuf,wbuf[0:cap(wbuf)],0,0,0,&stack{},0,0}
    p.parse()
    return wbuf[:p.wi]
}
func (p *parser) writeByte(by byte) {
        p.wbuf[p.wi] = by
        p.wi++
}
func (p *parser) write(by []byte) {
        p.wi += uint(copy(p.wbuf[p.wi:], by))
}
// func (p *parser) writeEntityEscaped(n int) {
//     s:=p.rbuf[p.ri:p.ri+n]
//     for i:=uint(0); i < uint(len(s)); i++ {
//         if entities[s[i]] == 1 {
//             if p.wi < uint(len(p.wbuf)) {
//                 p.wi += uint(copy(p.wbuf[p.wi:], s[:i]))
//             }
//             if p.wi < uint(len(p.wbuf)) {
//                 p.wi += uint(copy(p.wbuf[p.wi:], entitiesEnc[s[i]]))
//             }
//             s = s[i+1:]
//             i = 0
//         }
//     }
//     if p.wi < uint(len(p.wbuf)) {
//         p.wi += uint(copy(p.wbuf[p.wi:], s))
//     }
// }

var wb []byte =  make([]byte, 0, 512)
func (p *parser) writeEntityEscaped(n int) {
    if n == 0 {
        if p.wi < uint(len(p.wbuf)) && p.ri >=0 && p.ri < len(p.rbuf) {
            if entities[p.rbuf[p.ri]] != 1 {
                p.wi += uint(copy(p.wbuf[p.wi:], p.rbuf[p.ri:p.ri]))
            } else {
                p.wi += uint(copy(p.wbuf[p.wi:], entitiesEnc[p.rbuf[p.ri]]))
            }
        }
        return
    }

    s:=p.rbuf[p.ri:p.ri+n]
    wb:=wb[:0]
    o:=uint(0)
    m:=uint(len(s))
    // wb:=wb[:0]
    // for i:=uint(0); i < m ; i++ {
    for i := range m {
        if entities[s[i]] == 1 {
            if o < i {
                wb = append(append(wb, s[o:i]...), entitiesEnc[s[i]]...)
            } else {
                wb = append(wb, entitiesEnc[s[i]]...)
            }
            o = i+1
        }
    }
    if o < m && cap(wb) >= len(s[o:]) {
        // wb = append(wb, s[o:]...)
        wb = wb[0:len(s[o:])]
        copy(wb, s[o:])
    }
    if p.wi < uint(len(p.wbuf)) {
        p.wi += uint(copy(p.wbuf[p.wi:], wb))
    }
}

func (p *parser) openTagCond(t uint8) bool {
    for i:= p.si; i > 0; i-- {
        if p.stack[i].t == t {
            return false
        }
    }
    p.open(t)
    return true
}

func (p *parser) open(t uint8) {
    p.si++
    p.write(tags[t].open)
    p.stack[p.si] = tagNesting{t, p.indentation}
}

func (p *parser) close() {
    p.write(tags[p.stack[p.si].t].close)
    p.si--
}

// func (p *parser) openOrClose(t uint8) {
//     if p.hasTag(t) {
//         p.close()
//     } else {
//         p.open(t)
//     }
// }

func (p *parser) openOrClose(t uint8) {
    for i:= p.si; i > 0; i-- {
        if p.stack[i].t == t {
            p.wi += uint(copy(p.wbuf[p.wi:], tags[t].close))
            p.si--
            return
        }
    }
    p.si++
    p.wi += uint(copy(p.wbuf[p.wi:], tags[t].open))
    p.stack[p.si] = tagNesting{t, p.indentation}
}

func (p *parser) closeAll() {
    for p.si > 0 {
        p.close()
    }
}

func (p *parser) hasTag(t uint8) bool {
    for i:= p.si; i > 0; i-- {
        if p.stack[i].t == t {
            return true
        }
    }
    return false
}

func (p *parser) getTag(t uint8) tagNesting {
    for i:= p.si; i > 0; i-- {
        if p.stack[i].t == t {
            return p.stack[i]
        }
    }
    return tagNesting{0,0}
}

func (p *parser) getNestingLevel() uint8 {
    if p.si > 0 {
        return p.stack[p.si].level
    }
    return 0
}

func (p *parser) current() byte {
    if p.ri < p.rlen {
        return p.rbuf[p.ri]
    }
    return 0
}

func (p *parser) peek() byte {
    if p.ri + 1 < p.rlen {
        return p.rbuf[p.ri+1]
    }
    return 0
}

func (p *parser) skip(n int) {
    p.ri += n
}

func (p *parser) count(ch byte) int {
    i:=p.ri
    for ; i < p.rlen && p.rbuf[i] == ch; i++ {}
    return i - p.ri
}

func (p *parser) indexSyntax() int {
    i:=uint(0)
    s:=p.rbuf[p.ri:]
    k:=uint(len(s))
    for ; i < k && i+8 < k; i+=8 {
        if syntax[s[i+0]] == 1 {
            return int(i+0)
        } else if syntax[s[1]] == 1 {
            return int(i+1)
        } else if syntax[s[2]] == 1 {
            return int(i+2)
        } else if syntax[s[3]] == 1 {
            return int(i+3)
        } else if syntax[s[4]] == 1 {
            return int(i+4)
        } else if syntax[s[5]] == 1 {
            return int(i+5)
        } else if syntax[s[6]] == 1 {
            return int(i+6)
        } else if syntax[s[7]] == 1 {
            return int(i+7)
        }
    }
    for ; i < k; i++ {
        if syntax[s[i]] == 1 {
            return int(i)
        }
    }
    return -1
}

func (p *parser) eol() bool {
    if p.ri >= p.rlen {
        return true
    } else if c:=p.rbuf[p.ri]; c == '\n' || c == '\r' {
        return true
    }
    return false
}

func (p * parser) skipAndCountTabs() uint8 {
    if p.rbuf[p.ri] != '\t' && p.rbuf[p.ri] != ' ' {
        return 0
    }
    cnt:=uint8(0)
    for ; p.ri < p.rlen && (p.rbuf[p.ri] == '\t' || p.rbuf[p.ri] == ' '); p.ri++ {
        cnt += p.rbuf[p.ri]&1 * 3 + 1
    }
    return cnt/4
}

func (p * parser) skipAndCountNewlines() uint8 {
    cnt:=uint8(0)
    for ; p.ri < p.rlen && (p.rbuf[p.ri] == '\n' || p.rbuf[p.ri] == '\r'); p.ri++ {
        cnt += p.rbuf[p.ri]>>1&1
    }
    return cnt
}

func (p *parser) parse() {
    p.rlen = len(p.rbuf)

    /*
    todo:
    - links, references, footnotes
    - ordered lists
    - entities
    - escaping
    - images
    - todo more flexible rulers
    */
    var startOfLine bool = true
    var startOfBlock bool = true
    for p.ri < p.rlen {

        i:=p.indexSyntax()
        if i < 0 || p.ri + i >= p.rlen {
            p.write(p.rbuf[p.ri:])
            p.skip(len(p.rbuf[p.ri:]))
            break
        }
        p.write(p.rbuf[p.ri:p.ri+i])
        p.skip(i)

        startOfBlock = false
        startOfLine  = false
        if p.rbuf[p.ri] == '\n' || p.rbuf[p.ri] == '\r' {
            startOfLine = true
            if p.skipAndCountNewlines() > 1 {
                startOfBlock = true
                p.closeAll()
            } else {
                p.writeByte('\n')
            }
            if p.ri >= p.rlen {
                break
            }
            p.indentation = p.skipAndCountTabs()
        }

        current := p.current()

        switch {
        case startOfBlock && current == '#':
            cnt := p.count('#')
            p.skip(cnt)
            if cnt >= 1 && cnt <= 6 {
                p.open(uint8(cnt))
                continue
            }
            p.open(tagP)
            p.write(p.rbuf[p.ri-cnt:p.ri])
        case startOfBlock && current == '-':
            i:=p.count('-')
            p.skip(i)
            if i > 2 && p.eol() {
                p.write(tags[tagHr].close)
                p.writeByte('\n')
                continue
            } 
            p.open(tagP)
            p.write(p.rbuf[p.ri-i:p.ri])
        case startOfBlock && string(p.rbuf[p.ri:p.ri+3]) == "```":
            p.skip(3)
            if p.current() == '\n' {
                p.skip(1)
            }
            p.write(tags[tagPre].open)
            p.write(tags[tagCode].open)
            i := bytes.Index(p.rbuf[p.ri:], []byte("```"))
            if i < 0 {
                i = p.rlen - p.ri
            }
            p.writeEntityEscaped(i)
            p.skip(i+3)
            p.write(tags[tagCode].close)
            p.write(tags[tagPre].close)
        case startOfBlock && isLetter(current):
            p.openTagCond(tagP)
            p.writeByte(p.current())
            p.skip(1)

        case startOfLine && current == '*' && p.peek() == ' ':
            p.skip(2)
            if !p.hasTag(tagLi) {
                p.open(tagUl)
                p.open(tagLi)
                continue
            }
            level := p.getNestingLevel()
            switch {
            case p.indentation > level:
                p.open(tagUl)
                p.open(tagLi)
            case p.indentation == level:
                p.write([]byte("</li><li>"))
            case p.indentation < level:
                for p.stack[p.si].level > p.indentation {
                    p.close()
                }
                if p.hasTag(tagLi) {
                    p.write([]byte("</li><li>"))
                }
            }
        case startOfLine && current == '>':
            p.handleBlockquote()

        case current == '*' && p.peek() == '*':
            p.handleInlineTag(tagB)
        case current == '_' && p.peek() == '_':
            p.handleInlineTag(tagI)
        case current == '~' && p.peek() == '~':
            p.handleInlineTag(tagS)
        case current == '`':
            p.handleInlineCode()
        default:
            p.writeByte(current)
            p.skip(1)
        }

    }
    p.closeAll()
}

func (p * parser) indexByte(b byte) int {
    if i:=bytes.IndexByte(p.rbuf[p.ri:], b); i > -1 {
        return i
    } 
    return p.rlen - p.ri
}

func (p * parser) handleBlockquote() {
    level := int(p.getTag(tagBq).level)
    cnt := p.count('>')
    p.skip(cnt)
    switch {
    case level < cnt:
        for i:= level; i < cnt; i++ {
            p.si++
            p.write(tags[tagBq].open)
            p.stack[p.si] = tagNesting{tagBq, uint8(i + 1)}
        }
    case level > cnt:
        for int(p.stack[p.si].level) > cnt {
            p.close()
        }
    }
}

func (p * parser) handleInlineCode() {
    p.skip(1)
    p.write(tags[tagCode].open)
    i := p.indexByte('`')
    p.writeEntityEscaped(i)
    p.write(tags[tagCode].close)
    p.skip(i+1)
}

func (p * parser) handleInlineTag(t uint8) {
    p.skip(2)
    p.openOrClose(t)
}

func isLetter(c byte) bool {
    return (c >= 65 && c <= 90) || (c >= 97 && c <= 122)
}

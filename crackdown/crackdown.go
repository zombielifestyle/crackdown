
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

var tags = [256]tag {
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
}

func ConvertBytes(rbuf []byte, wbuf []byte) []byte {
    p := &parser{rbuf,wbuf[:0],0,sstack,0,0}
    p.parse()
    return p.wbuf
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

    p := &parser{rbuf,wbuf[:0],0,&stack{},0,0}
    p.parse()
    return p.wbuf
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

    p := &parser{rbuf,wbuf[:0],0,&stack{},0,0}
    p.parse()
    return p.wbuf
}

func (p *parser) writeByte(c byte) {
    p.wbuf = append(p.wbuf, c)
}

func (p *parser) write(s []byte) {
    p.wbuf = append(p.wbuf, s...)
}

func (p *parser) writeRange(i, j int) {
    if i >= 0 && j < len(p.rbuf) && i < j {
        p.write(p.rbuf[i:j])
    }
}

func (p *parser) writeEntityEscaped(s []byte) {
    o:=uint(0)
    for i:=uint(0); i < uint(len(s)); i++ {
        if entities[s[i]] == 0 {
            continue
        }
        if o < i {
            p.wbuf = append(append(p.wbuf, s[o:i]...), entitiesEnc[s[i]]...)
        } else {
            p.wbuf = append(p.wbuf, entitiesEnc[s[i]]...)
        }
        o = i+1
    }
    if o < uint(len(s)) {
        p.wbuf = append(p.wbuf, s[o:]...)
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

func (p *parser) openOrClose(t uint8) {
    for i:= p.si; i > 0; i-- {
        if p.stack[i].t == t {
            p.wbuf = append(p.wbuf, tags[t].close...)
            p.si--
            return
        }
    }
    p.si++
    p.wbuf = append(p.wbuf, tags[t].open...)
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
    return p.stack[0]
}

func (p *parser) getNestingLevel() uint8 {
    return p.stack[p.si].level
}

func (p *parser) current() byte {
    if p.ri >= 0 && p.ri < len(p.rbuf) {
        return p.rbuf[p.ri]
    }
    return 0
}

func (p *parser) peek() byte {
    if i:=p.ri+1; i >= 0 && i < len(p.rbuf){
        return p.rbuf[i]
    }
    return 0
}

func (p *parser) peekSlice(n int) string {
    if s,e:=p.ri,p.ri+n; s > 0 && e < len(p.rbuf) && s < e {
        return string(p.rbuf[s:e])
    }
    return ""
}

func (p *parser) skip(n int) {
    p.ri += n
}

func (p *parser) count(ch byte) int {
    i:=p.ri
    for ; i >= 0 && i < len(p.rbuf) && p.rbuf[i] == ch; i++ {}
    return i - p.ri
}

func (p *parser) indexSyntax() int {
    if p.ri < 0 || p.ri >= len(p.rbuf) {
        return -1
    }
    i:=uint(0)
    s:=p.rbuf[p.ri:]
    for ; uint(len(s)) > 7 ; i+=8 {
        if syntax[s[0]] == 1 {
            return int(i)
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
        s = s[8:]
    }
    for i:=0; i < len(s); i++ {
        if syntax[s[i]] == 1 {
            return int(i)
        }
    }
    return -1
}

func (p *parser) eol() bool {
    if p.ri < 0 || p.ri >= len(p.rbuf) {
        return true
    } else if c:=p.rbuf[p.ri]; c == '\n' || c == '\r' {
        return true
    }
    return false
}

func (p *parser) skipAndCountTabs() uint8 {
    cnt:=uint8(0)
    for ; p.ri >= 0 && p.ri < len(p.rbuf) && (p.rbuf[p.ri] == '\t' || p.rbuf[p.ri] == ' '); p.ri++ {
        cnt += p.rbuf[p.ri]&1 * 3 + 1
    }
    return cnt/4
}

func (p *parser) skipAndCountNewlines() uint8 {
    cnt:=uint8(0)
    for ; p.ri >= 0 && p.ri < len(p.rbuf) && (p.rbuf[p.ri] == '\n' || p.rbuf[p.ri] == '\r'); p.ri++ {
        cnt += p.rbuf[p.ri]>>1&1
    }
    return cnt
}

func (p *parser) parse() {
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
    for p.ri < len(p.rbuf) {

        i:=p.indexSyntax()
        if i < 0 || p.ri + i >= len(p.rbuf) {
            break
        }
        p.writeRange(p.ri, p.ri+i)
        p.skip(i)

        current := p.current()

        startOfBlock = false
        startOfLine  = false
        if current == '\n' || current == '\r' {
            startOfLine = true
            if p.skipAndCountNewlines() > 1 {
                startOfBlock = true
                p.closeAll()
            } else {
                p.writeByte('\n')
            }
            if p.ri >= len(p.rbuf) {
                break
            }
            p.indentation = p.skipAndCountTabs()
        }

        current = p.current()

        switch {
        case startOfBlock && current == '#':
            cnt := p.count('#')
            p.skip(cnt)
            if cnt >= 1 && cnt <= 6 {
                p.open(uint8(cnt))
                continue
            }
            p.open(tagP)
            p.writeRange(p.ri-cnt, p.ri)
        case startOfBlock && current == '-':
            i:=p.count('-')
            p.skip(i)
            if i > 2 && p.eol() {
                p.write(tags[tagHr].close)
                p.writeByte('\n')
                continue
            } 
            p.open(tagP)
            p.writeRange(p.ri-i, p.ri)
        case startOfBlock && p.peekSlice(3) == "```":
            p.skip(3)
            if p.current() == '\n' {
                p.skip(1)
            }
            p.write(tags[tagPre].open)
            p.write(tags[tagCode].open)
            s:=p.bytes()
            if i:=bytes.Index(s, []byte("```")); i > -1 && i < len(s) {
                p.writeEntityEscaped(s[:i])
                p.skip(i+3)
                p.write(tags[tagCode].close)
                p.write(tags[tagPre].close)
                continue
            }
            p.writeEntityEscaped(s)
            p.skip(len(s))
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
            p.skip(2)
            p.openOrClose(tagB)
        case current == '_' && p.peek() == '_':
            p.skip(2)
            p.openOrClose(tagI)
        case current == '~' && p.peek() == '~':
            p.skip(2)
            p.openOrClose(tagS)
        case current == '`':
            p.skip(1)
            p.write(tags[tagCode].open)
            s:=p.bytes()
            if i:=bytes.IndexByte(s, '`'); i > -1 && i < len(s) {
                p.writeEntityEscaped(s[:i])
                p.write(tags[tagCode].close)
                p.skip(i+1)
                continue
            }
            p.writeEntityEscaped(s)
            p.skip(len(s))
            p.write(tags[tagCode].close)

        default:
            p.writeByte(current)
            p.skip(1)
        }

    }
    if p.ri >= 0 && p.ri < len(p.rbuf) {
        p.write(p.rbuf[p.ri:])
    }
    p.closeAll()
}

func (p *parser) bytes() []byte {
    if p.ri >= 0 && p.ri < len(p.rbuf) {
        return p.rbuf[p.ri:]
    }
    return p.rbuf[:]
}

func (p *parser) handleBlockquote() {
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

func isLetter(c byte) bool {
    return (c >= 65 && c <= 90) || (c >= 97 && c <= 122)
}

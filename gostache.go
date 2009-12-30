package gostache

import (
    "bytes"
    "container/vector"
    "fmt"
    "io"
    "os"
    "reflect"
    "strings"
)

type textElement struct {
    text []byte
}

type varElement struct {
    name string
}

type sectionElement struct {
    name  string
    elems *vector.Vector
}

type template struct {
    data    string
    otag    string
    ctag    string
    p       int
    curline int
    elems   *vector.Vector
}

type parseError struct {
    line    int
    message string
}

func (p parseError) String() string { return fmt.Sprintf("line %d: %s", p.line, p.message) }

func (tmpl *template) readString(s string) (string, os.Error) {
    i := tmpl.p
    newlines := 0
    for true {
        //are we at the end of the string?
        if i+len(s) > len(tmpl.data) {
            return "", os.EOF
        }

        if tmpl.data[i] == '\n' {
            newlines++
        }

        if tmpl.data[i] != s[0] {
            i++
            continue
        }

        match := true
        for j := 1; j < len(s); j++ {
            if s[j] != tmpl.data[i+j] {
                match = false
                break
            }
        }

        if match {
            e := i + len(s)
            text := tmpl.data[tmpl.p:e]
            tmpl.p = e
            tmpl.curline += newlines
            return text, nil
        } else {
            i++
        }
    }

    //should never be here
    return "", nil
}

func (tmpl *template) readToEnd() string {
    text := tmpl.data[tmpl.p:]
    tmpl.p = len(tmpl.data)
    return text
}

func (tmpl *template) parseSection(section *sectionElement) os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            //put the remaining text in a block
            remaining := tmpl.readToEnd()
            section.elems.Push(&textElement{strings.Bytes(remaining)})
            return nil
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        section.elems.Push(&textElement{strings.Bytes(text)})

        text, err = tmpl.readString(tmpl.ctag)
        if err == os.EOF {
            //put the remaining text in a block
            return parseError{tmpl.curline, "unmatched open tag"}
        }

        //trim the close tag off the text
        tag := strings.TrimSpace(text[0 : len(text)-len(tmpl.ctag)])
        switch tag[0] {
        case '!':
            //ignore comment
            break
        case '#':
            name := strings.TrimSpace(tag[1:])
            se := sectionElement{name, new(vector.Vector)}
            err := tmpl.parseSection(&se)
            if err != nil {
                return err
            }
            section.elems.Push(&se)
        case '/':
            name := strings.TrimSpace(tag[1:])
            if name != section.name {
                return parseError{tmpl.curline, "interleaved closing tag: " + name}
            } else {
                return nil
            }
        case '>':
            break
        case '=':
            if tag[len(tag)-1] != '=' {
                panicln("Invalid meta tag")
            }
            tag = strings.TrimSpace(tag[1 : len(tag)-1])
            newtags := strings.Split(tag, " ", 0)
            if len(newtags) == 2 {
                tmpl.otag = newtags[0]
                tmpl.ctag = newtags[1]
            }
        default:
            section.elems.Push(&varElement{tag})
        }
    }

    return nil
}

func (tmpl *template) parse() os.Error {
    for {
        text, err := tmpl.readString(tmpl.otag)

        if err == os.EOF {
            //put the remaining text in a block
            remaining := tmpl.readToEnd()
            tmpl.elems.Push(&textElement{strings.Bytes(remaining)})
            return nil
        }

        // put text into an item
        text = text[0 : len(text)-len(tmpl.otag)]
        tmpl.elems.Push(&textElement{strings.Bytes(text)})

        text, err = tmpl.readString(tmpl.ctag)
        if err == os.EOF {
            //put the remaining text in a block
            return parseError{tmpl.curline, "unmatched open tag"}
        }

        //trim the close tag off the text
        tag := strings.TrimSpace(text[0 : len(text)-len(tmpl.ctag)])
        switch tag[0] {
        case '!':
            //ignore comment
            break
        case '#':
            name := strings.TrimSpace(tag[1:])
            se := sectionElement{name, new(vector.Vector)}
            err := tmpl.parseSection(&se)
            if err != nil {
                return err
            }
            tmpl.elems.Push(&se)
        case '/':
            return parseError{tmpl.curline, "unmatched close tag"}
        case '>':
            break
        case '=':
            if tag[len(tag)-1] != '=' {
                panicln("Invalid meta tag")
            }
            tag = strings.TrimSpace(tag[1 : len(tag)-1])
            newtags := strings.Split(tag, " ", 0)
            if len(newtags) == 2 {
                tmpl.otag = newtags[0]
                tmpl.ctag = newtags[1]
            }
        default:
            tmpl.elems.Push(&varElement{tag})
        }
    }

    return nil
}

func lookup(context reflect.Value, name string) reflect.Value {

    switch val := context.(type) {
    case *reflect.MapValue:
        return val.Elem(reflect.NewValue(name))
    case *reflect.StructValue:
        return val.FieldByName(name)
    }

    return nil
}

func executeSection(section *sectionElement, context reflect.Value, buf io.Writer) {
    value := lookup(context, section.name)

    switch val := value.(type) {
    case *reflect.BoolValue:
        if !val.Get() {
            return
        }
    }

    //by default we execute the section
    for i := 0; i < section.elems.Len(); i++ {
        executeElement(section.elems.At(i), context, buf)
    }

}

func executeElement(element interface{}, context reflect.Value, buf io.Writer) {
    switch elem := element.(type) {
    case *textElement:
        buf.Write(elem.text)
    case *varElement:
        val := lookup(context, elem.name)
        if val != nil {
            buf.Write(strings.Bytes(val.(*reflect.StringValue).Get()))
        }
    case *sectionElement:
        executeSection(elem, context, buf)
    }
}

func (tmpl *template) execute(context reflect.Value, buf io.Writer) {

    for i := 0; i < tmpl.elems.Len(); i++ {
        executeElement(tmpl.elems.At(i), context, buf)
    }

}

func Render(data string, context interface{}) (string, os.Error) {
    parser := template{data, "{{", "}}", 0, 1, new(vector.Vector)}
    err := parser.parse()

    if err != nil {
        return "", err
    }

    val := reflect.NewValue(context)
    var buf bytes.Buffer
    parser.execute(val, &buf)
    return buf.String(), nil
}
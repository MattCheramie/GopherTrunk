package radioreference

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// node is one parsed XML element: its local name (namespace prefix dropped),
// its own text, and its child elements in document order.
//
// The system import reads responses through this small tree instead of the
// first-match scanners in client.go (firstLeaves / blocks), because the
// RadioReference types nest elements of the same name at different depths: a
// site's <item> holds its frequencies' <item>s, and a Trs carries a <sysid>
// array whose entries each have a <sysid> field. Reading DIRECT children is
// the only way to keep those apart.
type node struct {
	name string
	text string
	kids []*node
}

// parseXML parses raw into a synthetic root whose children are the document's
// top-level elements.
func parseXML(raw []byte) (*node, error) {
	root := &node{}
	stack := []*node{root}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := dec.Token()
		if err != nil {
			if len(stack) != 1 {
				return nil, fmt.Errorf("radioreference: malformed XML response: %w", err)
			}
			return root, nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{name: t.Name.Local}
			top := stack[len(stack)-1]
			top.kids = append(top.kids, n)
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 1 {
				n := stack[len(stack)-1]
				n.text = strings.TrimSpace(n.text)
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			stack[len(stack)-1].text += string(t)
		}
	}
}

// find returns the first element named name at or below n, depth first, or nil.
func (n *node) find(name string) *node {
	if n == nil {
		return nil
	}
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
		if hit := k.find(name); hit != nil {
			return hit
		}
	}
	return nil
}

// child returns n's first DIRECT child named name, or nil.
func (n *node) child(name string) *node {
	if n == nil {
		return nil
	}
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	return nil
}

// val returns the text of n's direct child named name, "" when there is none.
func (n *node) val(name string) string {
	if c := n.child(name); c != nil {
		return c.text
	}
	return ""
}

// items returns the entries of a SOAP-encoded array. The service's PHP
// SoapServer encodes an rpc/encoded array as the array element carrying one
// <item> child per entry (issue #1197), so an empty array has no children.
func (n *node) items() []*node {
	if n == nil {
		return nil
	}
	var out []*node
	for _, k := range n.kids {
		if k.name == "item" {
			out = append(out, k)
		}
	}
	return out
}

// soapParam is one rpc/encoded request parameter. Parameters are sent in the
// order given, which must be the order of the WSDL's message parts.
type soapParam struct {
	name, xsdType, value string
}

func intParam(name string, v int) soapParam {
	return soapParam{name: name, xsdType: "xsd:int", value: fmt.Sprintf("%d", v)}
}

// callOp calls one operation with params followed by the authInfo block (the
// last part of every authenticated operation in the WSDL) and returns the
// response's <return> element, the single response part every operation
// declares.
func (c *Client) callOp(ctx context.Context, op string, params ...soapParam) (*node, error) {
	var b strings.Builder
	b.WriteString("<ns1:" + op + ">")
	for _, p := range params {
		fmt.Fprintf(&b, `<%s xsi:type="%s">%s</%s>`, p.name, p.xsdType, xmlEscape(p.value), p.name)
	}
	b.WriteString(c.authXML())
	b.WriteString("</ns1:" + op + ">")
	raw, err := c.call(ctx, b.String())
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, op)
	}
	doc, err := parseXML(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (%s)", err, op)
	}
	ret := doc.find("return")
	if ret == nil {
		return nil, fmt.Errorf("radioreference: %s: the response carries no <return> element (%d bytes)", op, len(raw))
	}
	return ret, nil
}

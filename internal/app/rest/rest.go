// Copyright (c) 2020 anatolygudkov. All rights reserved.
// Use of this source code is governed by MIT license
// that can be found in the LICENSE file.
package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Values contains values extracted from the request URI
// according to the routing configuration. For example, for the routing
// "/articles/:id" values contains value of the id.
type Values struct {
	values map[string]string
}

func newValues() *Values {
	return &Values{
		values: make(map[string]string),
	}
}

// Has returns true if the values contains a value for the specified name.
func (v *Values) Has(name string) bool {
	_, ok := v.values[name]
	return ok
}

// String returns string value for the specified name.
func (v *Values) String(name string) string {
	return v.values[name]
}

// Int returns int value for the specified name.
func (v *Values) Int(name string) (value int, err error) {
	val, ok := v.values[name]
	if !ok {
		return 0, fmt.Errorf("no value for the name: %s", name)
	}
	return strconv.Atoi(val)
}

// Dump dumps all values to an io.Writer.
func (v *Values) Dump(w io.Writer) {
	b, err := json.MarshalIndent(v.values, "", "  ")
	if err == nil {
		fmt.Fprint(w, string(b))
	}
}

// Handle handles http request for a route.
type Handle func(v *Values, res http.ResponseWriter, req *http.Request) error

// Srv is a REST server.
type Srv struct {
	addr      string
	trees     map[string]*tree
	treesLock sync.RWMutex
}

// NewSrv creates new instance of the Srv for the specified local address.
func NewSrv(addr string) *Srv {
	return &Srv{
		addr:  addr,
		trees: make(map[string]*tree),
	}
}

// Get registers new route for the HTTP GET requests.
func (s *Srv) Get(url string, handler Handle) {
	s.registerHandler(http.MethodGet, url, handler)
}

// Post registers new route for the HTTP POST requests.
func (s *Srv) Post(url string, handler Handle) {
	s.registerHandler(http.MethodPost, url, handler)
}

// Put registers new route for the HTTP PUT requests.
func (s *Srv) Put(url string, handler Handle) {
	s.registerHandler(http.MethodPut, url, handler)
}

// Delete registers new route for the HTTP DELETE requests.
func (s *Srv) Delete(url string, handler Handle) {
	s.registerHandler(http.MethodDelete, url, handler)
}

// Start starts the Srv.
func (s *Srv) Start() error {
	return http.ListenAndServe(s.addr, s)
}

// ServeHTTP implements http.Handler and routes incoming requests.
func (s *Srv) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var t *tree
	s.treesLock.RLock()
	func() {
		defer s.treesLock.RUnlock()
		t = s.trees[req.Method]
	}()

	if t == nil {
		httpError(res, http.StatusNotFound, fmt.Sprintf("Unmapped HTTP method: %s", req.Method))
		return
	}

	v, h, err := t.resolvePath(req.RequestURI)
	if err != nil || h == nil {
		httpError(res, http.StatusNotFound, fmt.Sprintf("URL %s not mapped", req.RequestURI))
		return
	}

	err = h(v, res, req)
	if err != nil {
		httpError(res, http.StatusInternalServerError, err)
		return
	}
}

func (s *Srv) registerHandler(httpMethod string, url string, handler Handle) {
	var t *tree
	s.treesLock.Lock()
	func() {
		defer s.treesLock.Unlock()
		t = s.trees[httpMethod]
		if t == nil {
			t = new(tree)
			s.trees[httpMethod] = t
		}
	}()
	t.applyPath(url, handler)
}

func httpError(res http.ResponseWriter, code int, cause interface{}) {
	res.WriteHeader(code)
	fmt.Fprintf(res, "An error: %v", cause)
}

type path struct {
	index          int
	url            string
	currentSegment *string
}

func newPath(url string) *path {
	return &path{
		index:          0,
		url:            strings.TrimSpace(url),
		currentSegment: nil,
	}
}

func (p *path) next() bool {
	p.currentSegment = nil

	if len(p.url) == 0 {
		return false
	}

	if len(p.url) == p.index {
		return false
	}

	var s strings.Builder

	for p.index < len(p.url) {
		switch c := p.url[p.index]; c {
		case '/':
			if p.index == 0 { // If root
				p.index++
				r := ""
				p.currentSegment = &r
				return true
			}
			r := strings.TrimSpace(s.String())
			if len(r) > 0 { // Not empty string collected
				p.index++
				p.currentSegment = &r
				return true
			}
			s.Reset()
			s.WriteString(r)
		default:
			s.WriteByte(c)
		}
		p.index++
	}

	r := strings.TrimSpace(s.String())
	if len(r) > 0 { // Not empty string collected
		p.currentSegment = &r
		return true
	}
	return false
}

func (p *path) segment() string {
	return *p.currentSegment
}

type nodeType int

const (
	constant nodeType = iota
	value
)

type node struct {
	segment  string
	nodeType nodeType
	handler  Handle
	next     map[string]*node
}

func newNode(s string) (n *node, err error) {
	nodeType := constant
	nodeSegment := s

	if len(s) > 0 {
		if s[0] == ':' {
			if len(s) < 2 {
				return nil, errors.New("invalid path segment ':'")
			}
			nodeType = value
			nodeSegment = s[1:]
		}
	}

	return &node{
		segment:  nodeSegment,
		nodeType: nodeType,
		handler:  nil,
		next:     nil,
	}, nil
}

func (n *node) applyNext(segment string) (rn *node, err error) {
	rn, err = newNode(segment)
	if err != nil {
		return nil, err
	}

	if n.next == nil {
		n.next = make(map[string]*node)
		n.next[rn.segment] = rn
		return rn, nil
	}

	nn := n.next[rn.segment]
	if nn != nil {
		if nn.nodeType != rn.nodeType {
			return nil, fmt.Errorf("ambiguous type of the segment '%s'", nn.segment)
		}
		rn = nn
		return rn, nil
	}

	n.next[rn.segment] = rn
	return rn, nil
}

type tree struct {
	root *node
}

func (t *tree) applyPath(path string, handler Handle) (err error) {
	p := newPath(path)

	if !p.next() {
		return errors.New("empty path")
	}

	n, err := newNode(p.segment())
	if err != nil {
		return err
	}

	if t.root == nil {
		t.root = n
	} else {
		if t.root.segment != p.segment() {
			return fmt.Errorf("incorrect root in the path: '%s'. Alreagy applied: '%s'", path, t.root.segment)
		}
	}

	n = t.root
	for p.next() {
		n, err = n.applyNext(p.segment())
		if err != nil {
			return err
		}
	}

	if n.handler != nil {
		return fmt.Errorf("multiple matched path: %s", path)
	}

	n.handler = handler

	return nil
}

func (t *tree) resolvePath(path string) (values *Values, handler Handle, err error) {
	n := t.root
	if n == nil {
		return nil, nil, errors.New("no any mapping exists")
	}

	next := make(map[string]*node)
	next[n.segment] = n

	p := newPath(path)

	values = newValues()
Search:
	for p.next() {
		s := p.segment()
		nn := next[s]

		if nn == nil {
			for _, v := range next {
				if v.nodeType == value {
					values.values[v.segment], _ = url.QueryUnescape(s)
					n = v
					next = v.next
					continue Search
				}
			}
			n = nil
			break Search
		}

		n = nn
		next = nn.next

		if nn.nodeType == value {
			values.values[nn.segment], _ = url.QueryUnescape(s)
			continue
		}
	}

	if n == nil {
		return nil, nil, errors.New("not matched")
	}
	if n.handler == nil {
		return nil, nil, errors.New("no associated handler found")
	}

	return values, n.handler, nil
}

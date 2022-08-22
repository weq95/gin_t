package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

type NumberInterface interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

type nodeType uint8

const maxParamCount uint8 = ^uint8(0)

const (
	static nodeType = iota
	root            //根路径
	param
	catchAll
)

func min[T NumberInterface](a, b T) T {
	if a <= b {
		return a
	}

	return b
}

// wildcardNum 计算通配符 ':' 和 '*' 的数量
func wildcardNum(path string) uint8 {
	var n uint

	for _, val := range []rune(path) {
		if val != ':' && val != '*' {
			continue
		}

		n += 1
	}

	if n >= uint(maxParamCount) {
		return maxParamCount
	}

	return uint8(n)
}

// routerNode 路由器节点
type routerNode struct {
	fullPath  string //完整路径
	path      string
	wildChild bool //是否有子节点
	nType     nodeType
	maxParams uint8
	priority  uint32 //权重
	indices   string
	children  []*routerNode
	handle    []*handleFuncNode
}

// incrementChildPrio 增加给定子节点的优先级并在必要时重新排序
func (r *routerNode) incrementChildPrio(idx int) int {
	r.children[idx].priority++

	var prio = r.children[idx].priority
	var newIdx = idx

	//权重如果小于当前已调整的权重, 则进行移动
	for newIdx > 0 && r.children[newIdx-1].priority < prio {
		//把上一项 和 当前项交换位置
		r.children[newIdx-1], r.children[newIdx] = r.children[newIdx], r.children[newIdx-1]

		newIdx--
	}

	if newIdx != idx {
		r.indices = r.indices[:newIdx] + //r.indices = r.indices[:0]
			r.indices[newIdx:idx+1] +
			r.indices[newIdx:idx] +
			r.indices[idx+1:]
	}

	return newIdx
}

// addRoute adds a routerNode with the given handle to the path.
// Not concurrency-safe!
func (n *routerNode) addRoute(path string, handle []*handleFuncNode) {
	n.fullPath = path
	n.priority++

	//计算路径中通配符的数量
	var wildcard = wildcardNum(path)

	//还没有注册过路径 Empty tree
	if len(n.path) == 0 && len(n.children) == 0 {
		n.insertChild(wildcard, path, n.fullPath, handle)
		n.nType = root
		return
	}

	// non-empty tree
walk:
	for {
		// Update maxParams of the current routerNode
		if wildcard > n.maxParams {
			n.maxParams = wildcard
		}

		// Find the longest common prefix.
		// This also implies that the common prefix contains no ':' or '*'
		// since the existing key can't contain those chars.
		i := 0
		max := min(len(path), len(n.path))
		for i < max && path[i] == n.path[i] {
			i++
		}

		// Split edge
		if i < len(n.path) {
			child := routerNode{
				path:      n.path[i:],
				wildChild: n.wildChild,
				nType:     static,
				indices:   n.indices,
				children:  n.children,
				handle:    n.handle,
				priority:  n.priority - 1,
			}

			// Update maxParams (max of all children)
			for i := range child.children {
				if child.children[i].maxParams > child.maxParams {
					child.maxParams = child.children[i].maxParams
				}
			}

			n.children = []*routerNode{&child}
			// []byte for proper unicode char conversion, see #65
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handle = nil
			n.wildChild = false
		}

		if i == len(path) { // Make routerNode a (in-path) leaf
			if n.handle != nil {
				panic("a handle is already registered for path '" + n.fullPath + "'")
			}
			n.handle = handle
		}

		if i > len(path) {
			return
		}

		// Make new routerNode a child of this routerNode
		path = path[i:]

		if n.wildChild {
			n = n.children[0]
			n.priority++

			// Update maxParams of the child routerNode
			if wildcard > n.maxParams {
				n.maxParams = wildcard
			}
			wildcard--

			// Check if the wildcard matches
			if len(path) >= len(n.path) && n.path == path[:len(n.path)] &&
				// Adding a child to a catchAll is not possible
				n.nType != catchAll &&
				// Check for longer wildcard, e.g. :name and :names
				(len(n.path) >= len(path) || path[len(n.path)] == '/') {
				continue walk
			} else {
				// Wildcard conflict
				var pathSeg string
				if n.nType == catchAll {
					pathSeg = path
				} else {
					pathSeg = strings.SplitN(path, "/", 2)[0]
				}
				prefix := n.fullPath[:strings.Index(n.fullPath, pathSeg)] + n.path
				panic("'" + pathSeg +
					"' in new path '" + n.fullPath +
					"' conflicts with existing wildcard '" + n.path +
					"' in existing prefix '" + prefix +
					"'")
			}
		}

		c := path[0]

		// slash after param
		if n.nType == param && c == '/' && len(n.children) == 1 {
			n = n.children[0]
			n.priority++
			continue walk
		}

		// Check if a child with the next path byte exists
		for i := 0; i < len(n.indices); i++ {
			if c == n.indices[i] {
				i = n.incrementChildPrio(i)
				n = n.children[i]
				continue walk
			}
		}

		// Otherwise insert it
		if c != ':' && c != '*' {
			// []byte for proper unicode char conversion, see #65
			n.indices += string([]byte{c})
			child := &routerNode{
				maxParams: wildcard,
			}
			n.children = append(n.children, child)
			n.incrementChildPrio(len(n.indices) - 1)
			n = child
		}
		n.insertChild(wildcard, path, n.fullPath, handle)
		break
	}
}

func (n *routerNode) insertChild(wildcard uint8, path, fullPath string, handle []*handleFuncNode) {
	var offset int // already handled bytes of the path

	//路径中没有通配符
	if wildcard == 0 {
		// insert remaining path part and handle to the leaf
		n.path = path[offset:]
		n.handle = handle

		return
	}

	// find prefix until first wildcard (beginning with ':'' or '*'')
	for i, max := 0, len(path); wildcard > 0; i++ {
		//不是通配符就跳过
		if path[i] != ':' && path[i] != '*' {
			continue
		}

		var c = path[i]

		// find wildcard end (either '/' or path end)
		end := i + 1
		for end < max && path[end] != '/' {
			switch path[end] {
			// the wildcard name must not contain ':' and '*'
			case ':', '*':
				panic("only one wildcard per path segment is allowed, has: '" +
					path[i:] + "' in path '" + fullPath + "'")
			default:
				end++
			}
		}

		// check if this Node existing children which would be
		// unreachable if we insert the wildcard here
		if len(n.children) > 0 {
			panic("wildcard route '" + path[i:end] +
				"' conflicts with existing children in path '" + fullPath + "'")
		}

		// check if the wildcard has a name
		if end-i < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		if c == ':' { // param
			// split path at the beginning of the wildcard
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			child := &routerNode{
				nType:     param,
				maxParams: wildcard,
			}
			n.children = []*routerNode{child}
			n.wildChild = true
			n = child
			n.priority++
			wildcard--

			// if the path doesn't end with the wildcard, then there
			// will be another non-wildcard subpath starting with '/'
			if end < max {
				n.path = path[offset:end]
				offset = end

				child := &routerNode{
					maxParams: wildcard,
					priority:  1,
				}
				n.children = []*routerNode{child}
				n = child
			}

		} else { // catchAll
			if end != max || wildcard > 1 {
				panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			// currently fixed width 1 for '/'
			i--
			if path[i] != '/' {
				panic("no / before catch-all in path '" + fullPath + "'")
			}

			n.path = path[offset:i]

			// first routerNode: catchAll routerNode with empty path
			child := &routerNode{
				wildChild: true,
				nType:     catchAll,
				maxParams: 1,
			}
			// update maxParams of the parent routerNode
			if n.maxParams < 1 {
				n.maxParams = 1
			}
			n.children = []*routerNode{child}
			n.indices = string(path[i])
			n = child
			n.priority++

			// second routerNode: routerNode holding the variable
			child = &routerNode{
				path:      path[i:],
				nType:     catchAll,
				maxParams: 1,
				handle:    handle,
				priority:  1,
			}
			n.children = []*routerNode{child}

			return
		}
	}

}

// Returns the handle registered with the given path (key). The values of
// wildcards are saved to a map.
// If no handle can be found, a TSR (trailing slash redirect) recommendation is
// made if a handle exists with an extra (without the) trailing slash for the
// given path.
func (n *routerNode) getValue(path string) (handle []*handleFuncNode, p Params, tsr bool) {
walk: // outer loop for walking the tree
	for {
		if len(path) > len(n.path) {
			if path[:len(n.path)] == n.path {
				path = path[len(n.path):]
				// If this routerNode does not have a wildcard (param or catchAll)
				// child,  we can just look up the next child routerNode and continue
				// to walk down the tree
				if !n.wildChild {
					c := path[0]
					for i := 0; i < len(n.indices); i++ {
						if c == n.indices[i] {
							n = n.children[i]
							continue walk
						}
					}

					// Nothing found.
					// We can recommend to redirect to the same URL without a
					// trailing slash if a leaf exists for that path.
					tsr = path == "/" && n.handle != nil
					return

				}

				// handle wildcard child
				n = n.children[0]
				switch n.nType {
				case param:
					// find param end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// save param value
					if p == nil {
						// lazy allocation
						p = make(Params, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1] // expand slice within preallocated capacity
					p[i].Key = n.path[1:]
					p[i].Val = path[:end]

					// we need to go deeper!
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						// ... but we can't
						tsr = len(path) == end+1
						return
					}

					if handle = n.handle; handle != nil {
						return
					} else if len(n.children) == 1 {
						// No handle found. Check if a handle for this path + a
						// trailing slash exists for TSR recommendation
						n = n.children[0]
						tsr = n.path == "/" && n.handle != nil
					}

					return

				case catchAll:
					// save param value
					if p == nil {
						// lazy allocation
						p = make(Params, 0, n.maxParams)
					}
					i := len(p)
					p = p[:i+1] // expand slice within preallocated capacity
					p[i].Key = n.path[2:]
					p[i].Val = path

					handle = n.handle
					return

				default:
					panic("invalid routerNode type")
				}
			}
		} else if path == n.path {
			// We should have reached the routerNode containing the handle.
			// Check if this routerNode has a handle registered.
			if handle = n.handle; handle != nil {
				return
			}

			if path == "/" && n.wildChild && n.nType != root {
				tsr = true
				return
			}

			// No handle found. Check if a handle for this path + a
			// trailing slash exists for trailing slash recommendation
			for i := 0; i < len(n.indices); i++ {
				if n.indices[i] == '/' {
					n = n.children[i]
					tsr = (len(n.path) == 1 && n.handle != nil) ||
						(n.nType == catchAll && n.children[0].handle != nil)
					return
				}
			}

			return
		}

		// Nothing found. We can recommend to redirect to the same URL with an
		// extra trailing slash if a leaf exists for that path
		tsr = (path == "/") ||
			(len(n.path) == len(path)+1 && n.path[len(path)] == '/' &&
				path == n.path[:len(n.path)-1] && n.handle != nil)
		return
	}
}

// 对给定路径进行不区分大小写的查找并尝试查找处理程序。它还可以选择修复尾部斜杠。它返回大小写更正的路径和一个指示查找是否成功的布尔值。
func (r *routerNode) findCaseInsensitivePath(path string, fix bool) ([]byte, bool) {
	return r.fpRec(path, make([]byte, 0, len(path)+1), [4]byte{}, fix)
}

// recursive case-insensitive lookup function used by n.findCaseInsensitivePath
func (n *routerNode) fpRec(path string, ciPath []byte, rb [4]byte, fixTrailingSlash bool) ([]byte, bool) {
	npLen := len(n.path)

walk: // outer loop for walking the tree
	for len(path) >= npLen && (npLen == 0 || strings.EqualFold(path[1:npLen], n.path[1:])) {
		// add common prefix to result

		oldPath := path
		path = path[npLen:]
		ciPath = append(ciPath, n.path...)

		if len(path) > 0 {
			// If this routerNode does not have a wildcard (param or catchAll) child,
			// we can just look up the next child routerNode and continue to walk down
			// the tree
			if !n.wildChild {
				// skip rune bytes already processed
				rb = shiftNRuneBytes(rb, npLen)

				if rb[0] != 0 {
					// old rune not finished
					for i := 0; i < len(n.indices); i++ {
						if n.indices[i] == rb[0] {
							// continue with child routerNode
							n = n.children[i]
							npLen = len(n.path)
							continue walk
						}
					}
				} else {
					// process a new rune
					var rv rune

					// find rune start
					// runes are up to 4 byte long,
					// -4 would definitely be another rune
					var off int
					for max := min(npLen, 3); off < max; off++ {
						if i := npLen - off; utf8.RuneStart(oldPath[i]) {
							// read rune from cached path
							rv, _ = utf8.DecodeRuneInString(oldPath[i:])
							break
						}
					}

					// calculate lowercase bytes of current rune
					lo := unicode.ToLower(rv)
					utf8.EncodeRune(rb[:], lo)

					// skip already processed bytes
					rb = shiftNRuneBytes(rb, off)

					for i := 0; i < len(n.indices); i++ {
						// lowercase matches
						if n.indices[i] == rb[0] {
							// must use a recursive approach since both the
							// uppercase byte and the lowercase byte might exist
							// as an index
							if out, found := n.children[i].fpRec(
								path, ciPath, rb, fixTrailingSlash,
							); found {
								return out, true
							}
							break
						}
					}

					// if we found no match, the same for the uppercase rune,
					// if it differs
					if up := unicode.ToUpper(rv); up != lo {
						utf8.EncodeRune(rb[:], up)
						rb = shiftNRuneBytes(rb, off)

						for i, c := 0, rb[0]; i < len(n.indices); i++ {
							// uppercase matches
							if n.indices[i] == c {
								// continue with child routerNode
								n = n.children[i]
								npLen = len(n.path)
								continue walk
							}
						}
					}
				}

				// Nothing found. We can recommend to redirect to the same URL
				// without a trailing slash if a leaf exists for that path
				return ciPath, (fixTrailingSlash && path == "/" && n.handle != nil)
			}

			n = n.children[0]
			switch n.nType {
			case param:
				// find param end (either '/' or path end)
				k := 0
				for k < len(path) && path[k] != '/' {
					k++
				}

				// add param value to case insensitive path
				ciPath = append(ciPath, path[:k]...)

				// we need to go deeper!
				if k < len(path) {
					if len(n.children) > 0 {
						// continue with child routerNode
						n = n.children[0]
						npLen = len(n.path)
						path = path[k:]
						continue
					}

					// ... but we can't
					if fixTrailingSlash && len(path) == k+1 {
						return ciPath, true
					}
					return ciPath, false
				}

				if n.handle != nil {
					return ciPath, true
				} else if fixTrailingSlash && len(n.children) == 1 {
					// No handle found. Check if a handle for this path + a
					// trailing slash exists
					n = n.children[0]
					if n.path == "/" && n.handle != nil {
						return append(ciPath, '/'), true
					}
				}
				return ciPath, false

			case catchAll:
				return append(ciPath, path...), true

			default:
				panic("invalid routerNode type")
			}
		} else {
			// We should have reached the routerNode containing the handle.
			// Check if this routerNode has a handle registered.
			if n.handle != nil {
				return ciPath, true
			}

			// No handle found.
			// Try to fix the path by adding a trailing slash
			if fixTrailingSlash {
				for i := 0; i < len(n.indices); i++ {
					if n.indices[i] == '/' {
						n = n.children[i]
						if (len(n.path) == 1 && n.handle != nil) ||
							(n.nType == catchAll && n.children[0].handle != nil) {
							return append(ciPath, '/'), true
						}
						return ciPath, false
					}
				}
			}
			return ciPath, false
		}
	}

	// Nothing found.
	// Try to fix the path by adding / removing a trailing slash
	if fixTrailingSlash {
		if path == "/" {
			return ciPath, true
		}
		if len(path)+1 == npLen && n.path[len(path)] == '/' &&
			strings.EqualFold(path[1:], n.path[1:len(path)]) && n.handle != nil {
			return append(ciPath, n.path...), true
		}
	}
	return ciPath, false
}

// shiftNRuneBytes 将数组的字节向左移动 N 个字节
func shiftNRuneBytes(rb [4]byte, n int) [4]byte {
	switch n {
	case 0:
		return rb
	case 1:
		return [4]byte{rb[1], rb[2], rb[3]}
	case 2:
		return [4]byte{rb[2], rb[3]}
	case 3:
		return [4]byte{rb[3]}
	default:
		return [4]byte{}
	}
}

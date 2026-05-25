package livereload

import _ "embed"

// ClientJS is the default browser-side reload script. It connects to the
// hub over WebSocket (with EventSource fallback) and reloads the page on
// any incoming message. Consumers wanting smarter behavior (e.g.
// in-place DOM swap) can supply their own script via WithClientJS.
//
//go:embed client.js
var ClientJS string

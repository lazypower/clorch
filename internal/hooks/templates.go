package hooks

import _ "embed"

//go:embed event_handler.sh.tmpl
var eventHandlerTmpl string

//go:embed notify_handler.sh.tmpl
var notifyHandlerTmpl string

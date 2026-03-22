package tui

func renderFooter(yolo bool, sound bool, hooksStale bool) string {
	line := footerStyle.Render("j/k:navigate  →:jump  y/n:approve  Y:all  i:inject  b:branch  l:label  W:rename  !:yolo  s:sound  d:detail  ?:help  q:quit")
	if hooksStale {
		line += "  " + staleHookStyle.Render("⚠ hooks outdated — run clorch init")
	}
	return line
}

package tui

func renderFooter(yolo bool, sound bool) string {
	return footerStyle.Render("j/k:navigate  →:jump  y/n:approve  Y:all  !:yolo  s:sound  d:detail  ?:help  q:quit")
}

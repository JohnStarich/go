package main

const (
	godocHTML = `<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="theme-color" content="#375EAB">
{{with .Tabtitle}}
  <title>{{html .}} - {{gopages "GoPages" "SiteTitle"}}</title>
{{else}}
  <title>{{gopages "GoPages" "SiteTitle"}}</title>
{{end}}
<link type="text/css" rel="stylesheet" href="{{gopages "" "BaseURL"}}/lib/godoc/style.css">
{{if .TreeView}}
<link rel="stylesheet" href="{{gopages "" "BaseURL"}}/lib/godoc/jquery.treeview.css">
{{end}}
<script>window.initFuncs = [];</script>
<script src="{{gopages "" "BaseURL"}}/lib/godoc/jquery.js" defer></script>
{{if .TreeView}}
<script src="{{gopages "" "BaseURL"}}/lib/godoc/jquery.treeview.js" defer></script>
<script src="{{gopages "" "BaseURL"}}/lib/godoc/jquery.treeview.edit.js" defer></script>
{{end}}

{{if .Playground}}
<script src="{{gopages "" "BaseURL"}}/lib/godoc/playground.js" defer></script>
{{end}}
{{with .Version}}<script>var goVersion = {{printf "%q" .}};</script>{{end}}
<script src="{{gopages "" "BaseURL"}}/lib/godoc/godocs.js" defer></script>
</head>
<body>

<div id='lowframe' style="position: fixed; bottom: 0; left: 0; height: 0; width: 100%; border-top: thin solid grey; background-color: white; overflow: auto;">
...
</div><!-- #lowframe -->

<div id="topbar"{{if .Title}} class="wide"{{end}}><div class="container">
<div class="top-heading" id="heading-wide"><a href="{{gopages "" "BaseURL"}}/pkg/">{{gopages "GoPages | Auto-generated docs" "SiteTitleLong" "SiteTitle"}}</a></div>
<div class="top-heading" id="heading-narrow"><a href="{{gopages "" "BaseURL"}}/pkg/">{{gopages "GoPages" "SiteTitle"}}</a></div>
<a href="#" id="menu-button"><span id="menu-button-arrow">&#9661;</span></a>

</div></div>

{{if .Playground}}
<div id="playground" class="play">
	<div class="input"><textarea class="code" spellcheck="false">package main

import "fmt"

func main() {
	fmt.Println("Hello, 世界")
}</textarea></div>
	<div class="output"></div>
	<div class="buttons">
		<a class="run" title="Run this code [shift-enter]">Run</a>
		<a class="fmt" title="Format this code">Format</a>
		{{if not $.GoogleCN}}
		<a class="share" title="Share this code">Share</a>
		{{end}}
	</div>
</div>
{{end}}

<div id="page"{{if .Title}} class="wide"{{end}}>
<div class="container">

{{if or .Title .SrcPath}}
  <h1>
    {{html .Title}}
    {{html .SrcPath | srcBreadcrumb}}
  </h1>
{{end}}

{{with .Subtitle}}
  <h2>{{html .}}</h2>
{{end}}

{{with .SrcPath}}
  <h2>
    Documentation: {{html . | srcToPkgLink}}
  </h2>
{{end}}

{{/* The Table of Contents is automatically inserted in this <div>.
     Do not delete this <div>. */}}
<div id="nav"></div>

{{/* Body is HTML-escaped elsewhere */}}
{{printf "%s" .Body}}

<div id="footer">
Build version {{html .Version}}.<br>
</div>

</div><!-- .container -->
</div><!-- #page -->
</body>
</html>
`
)

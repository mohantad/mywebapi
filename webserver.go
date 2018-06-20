//
// Tiny web server in Golang for sharing a folder
// Based on Copyright (c) 2010-2014 Alexis ROBERT <alexis.robert@gmail.com>
// Modified by MOHANTA, Deepak <deepak.mohanta@outlook.com>
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// TODO: Find a way to be cleaner
var root_folder *string

const servername = "Tiny Web Server"

func main() {
	// Get current working directory to get the file from it
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error while getting current directory.")
		return
	}

	// Command line parsing
	bind := flag.String("port", ":8091", "bind address")
	root_folder = flag.String("folder", cwd, "htdocs root folder")

	flag.Parse()

	http.Handle("/", http.HandlerFunc(handleFile))

	fmt.Printf("Publishing folder: %s  on port%s ...\n", *root_folder, *bind)
	http.ListenAndServe((*bind), nil)
}

//-----------------------------------------------------------------------------
// DATA STRUCTURES
//-----------------------------------------------------------------------------

type Entry struct {
	Name    string
	Size    int64
	SizeStr string
	Time    string
	TimeUtc string
	ModTime time.Time
}

type DirListing struct {
	Name        string
	Directories []Entry
	Files       []Entry
	CntDirs     int
	CntFiles    int
}

//-----------------------------------------------------------------------------
// SORTING
//-----------------------------------------------------------------------------

// Sort files by name, in the A till Z order
type ByName []Entry

// Sort files by time, latest first
type ByTime []Entry

// Sort files by Size, biggest first
type BySize []Entry

func (a ByName) Len() int {
	return len(a)
}

func (a ByName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByName) Less(i, j int) bool {
	res := strings.Compare(a[i].Name, a[j].Name)
	return res == -1
}

func (a ByTime) Len() int {
	return len(a)
}

func (a ByTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByTime) Less(i, j int) bool {
	return a[i].ModTime.Unix() > a[j].ModTime.Unix()
}

func (a BySize) Len() int {
	return len(a)
}

func (a BySize) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a BySize) Less(i, j int) bool {
	return a[i].Size > a[j].Size
}

//-----------------------------------------------------------------------------

func sizeConv(size int64) string {
	var buf bytes.Buffer
	var comma byte = 44

	number := strconv.FormatInt(size, 10)
	l := len(number)

	// no rune business because numbers fit one byte
	for pos, _ := range number {
		if (l-pos)%3 == 0 && pos != 0 {
			buf.WriteByte(comma)
		}
		buf.WriteByte(number[pos])
	}

	return buf.String()
}

// TODO: path!!! the end at the path url for directories

func pathLink(path string) string {
	link := "<a href=\"/\">&lt;root&gt;</a>"
	current := "/"
	dirs := strings.Split(path, "/")

	for _, dir := range dirs[1:] {
		current = current + dir + "/"
		link = link + "/"
		link = link + "<a href=\""
		link = link + current
		link = link + "\">"
		link = link + dir
		link = link + "</a>"
	}

	return link
}

//-----------------------------------------------------------------------------

func handleDirectory(f *os.File, w http.ResponseWriter, req *http.Request) {
	names, _ := f.Readdir(-1)

	// First, check if there is any index in this folder.
	for _, val := range names {
		if val.Name() == "index.html" || val.Name() == "index.htm" {
			serveFile(path.Join(f.Name(), "index.html"), w, req)
			return
		}
	}

	// Otherwise, generate folder content.

	dirs := make([]Entry, 0)
	files := make([]Entry, 0)

	for _, val := range names {

		// Remove hidden files from listing
		// if val.Name()[0] == '.' {
		// continue
		// }

		timefmt := "2006-01-02 15:04:05 MST"

		entry := Entry{
			Name:    val.Name(),
			Size:    val.Size(),
			SizeStr: sizeConv(val.Size()),
			Time:    val.ModTime().Format(timefmt),
			TimeUtc: val.ModTime().UTC().Format(timefmt),
			ModTime: val.ModTime(),
		}

		if val.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	tpl, err := template.New("tpl").Parse(DirListingTmpl)
	if err != nil {
		http.Error(w, "Internal Oops!.", 500)
		fmt.Println(err)
		return
	}

	if strings.HasSuffix(req.RequestURI, "?t") {
		sort.Sort(ByTime(dirs))
		sort.Sort(ByTime(files))
	} else if strings.HasSuffix(req.RequestURI, "?s") {
		sort.Sort(BySize(dirs))
		sort.Sort(BySize(files))
	} else {
		sort.Sort(ByName(dirs))
		sort.Sort(ByName(files))
	}

	data := DirListing{
		Name:        pathLink(req.URL.Path),
		Directories: dirs,
		Files:       files,
		CntDirs:     len(dirs),
		CntFiles:    len(files),
	}

	err = tpl.Execute(w, data)
	if err != nil {
		fmt.Println(err)
	}
}

func serveFile(filepath string, w http.ResponseWriter, req *http.Request) {
	// Opening the file handle
	f, err := os.Open(filepath)
	if err != nil {
		http.Error(w, "Oops!", 404)
		return
	}

	defer f.Close()

	// Checking if the opened handle is really a file
	statinfo, err := f.Stat()
	if err != nil {
		http.Error(w, "500 Internal Oops! Take two.", 500)
		return
	}

	// If it's a directory, open it
	if statinfo.IsDir() {
		handleDirectory(f, w, req)
		return
	}

	// If it's a socket, forbid it
	if (statinfo.Mode() &^ 07777) == os.ModeSocket {
		http.Error(w, "Forbidden Oops!", 403)
		return
	}

	// Manages If-Modified-Since and add Last-Modified (taken from Golang code)
	t, err := time.Parse(http.TimeFormat, req.Header.Get("If-Modified-Since"))
	if err == nil && statinfo.ModTime().Unix() <= t.Unix() {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Last-Modified", statinfo.ModTime().Format(http.TimeFormat))

	// Fetching file's mimetype and giving it to the browser
	if mimetype := mime.TypeByExtension(path.Ext(filepath)); mimetype != "" {
		w.Header().Set("Content-Type", mimetype)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	// Add Content-Length
	w.Header().Set("Content-Length", strconv.FormatInt(statinfo.Size(), 10))

	_, err = io.Copy(w, f)
	if err != nil {
		http.Error(w, "File Oops!", 404)
	}
}

func handleFile(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Server", servername)

	filepath := path.Join((*root_folder), path.Clean(req.URL.Path))
	serveFile(filepath, w, req)
}

// TODO: look what parameters to pass to templates (type for data)

const DirListingTmpl = `<?xml version="1.0" encoding="iso-8859-1"?>
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="en">
<!-- Modified from lighttpd directory listing -->
<head>
<title>EFF</title>
<style type="text/css">
a, a:active {text-decoration: none; color: blue;}
a:visited {color: #48468F;}
a:hover, a:focus {text-decoration: underline; color: red;}

a.header, a.header:active, a.header:visited, a.header:hover, a.header:focus {text-decoration: none; color: black;}

body {background-color: #F5F5F5;}
h4 {margin-bottom: 12px;}
table {margin-left: 12px;}
th, td { font: 90% monospace; text-align: left;}
th.size, td.size { text-align: right;}
th { font-weight: bold; padding-right: 14px; padding-bottom: 3px;}
td {padding-right: 14px;}
td.s, th.s {text-align: right;}
div.list { background-color: white; border-top: 1px solid #646464; border-bottom: 1px solid #646464; padding-top: 10px; padding-bottom: 14px;}
div.foot { font: 90% monospace; color: #787878; padding-top: 4px;}
</style>
</head>
<body>

<h5></h5>
<table cellpadding="0" cellspacing="0">
<tr>
	<td>
	<font size="5"><bold>Tagbox server logs:  {{.Name}}</bold></font><br/><br/>Folders: {{.CntDirs}}&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;Files: {{.CntFiles}}<br/>
	</td>
</tr>
</table>

<div class="list">
<table summary="Directory Listing" cellpadding="0" cellspacing="0">
<thead>
<tr>
<th class="name"><a class="header" href=".">Name</a></th>
<th class="time"><a class="header" href=".?t">Time</a></th>
<th class="size"><a class="header" href=".?s">Size</a></th>
</tr>
</thead>

<tbody>

<tr>
<td colspan="3">&nbsp;</td>
</tr>

<tr>
<td class="name"><a href="../">..</a>/</td>
<td class="time"></td>
<td class="size"></td>
</tr>

{{range .Directories}}
<tr>
<td class="name"><a href="./{{.Name}}/">{{.Name}}/</a></td>
<td class="time">{{.Time}}  / {{.TimeUtc}}</td>
<td class="size">{{.SizeStr}}</td>
</tr>
{{end}}

{{if .Files}}
<tr>
<td colspan="3">&nbsp;</td>
</tr>
{{end}}

{{range .Files}}
<tr>
<td class="name"><a href="{{urlquery .Name}}">{{html .Name}}</a></td>
<td class="time">{{.Time}}  / {{.TimeUtc}}</td>
<td class="size">{{.SizeStr}}</td>
</tr>
{{end}}

</tbody>
</table>

</div>

</body>
</html>`

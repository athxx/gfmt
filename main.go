package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var stdLibs = map[string]int{"archive": 0, "tar": 0, "zip": 0, "bufio": 0, "builtin": 0, "bytes": 0, "compress": 0, "container": 0, "context": 0, "crypto": 0, "database": 0, "debug": 0, "embed": 0, "encoding": 0, "errors": 0, "expvar": 0, "flag": 0, "fmt": 0, "go": 0, "hash": 0, "html": 0, "image": 0, "index": 0, "io": 0, "log": 0, "math": 0, "mime": 0, "net": 0, "os": 0, "path": 0, "plugin": 0, "reflect": 0, "regexp": 0, "runtime": 0, "sort": 0, "strconv": 0, "strings": 0, "sync": 0, "syscall": 0, "testing": 0, "text": 0, "time": 0, "unicode": 0, "unsafe": 0, "internal": 0}
var separator = "\n"

type Pack struct {
	Name       string
	Alias      string
	Annotation string
}

var (
	impC   string
	impStd = map[string][]Pack{}
	impPrj = map[string][]Pack{}
	impExt = map[string][]Pack{}
	impAno []string
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("\nUsage : gfmt [path]")
	}
	s := os.Args[1]
	_, err := os.Stat(s) //os.Stat获取文件信息
	if err != nil {
		log.Fatalln("\nfile or directory not exist")
	}
	execGofmt(s)
	files := getFileList(s)

	for _, v := range files {
		rewriteFile(v)
	}
}

// get *.go file list
func getFileList(dirPath string) (f []string) {
	filepath.Walk(dirPath,
		func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if p, err = filepath.Abs(p); err != nil {
				return nil
			}
			if !info.IsDir() && filepath.Ext(p) == ".go" && info.Size() > 21 {
				f = append(f, p)
			}
			return nil
		})
	return f
}

func rewriteFile(file string) {
	buf, _ := ioutil.ReadFile(file)
	if strings.Contains(string(buf), "\r\n") {
		separator = "\r\n"
	}
	txt := strings.TrimSpace(string(buf))
	if txt[:8] != "package " {
		return
	}
	txt += separator // add a final separator
	// not match import
	r, _ := regexp.Compile(`\r?\nimport[\s\t\r\n]+`)
	if !r.MatchString(txt) {
		return
	}

	// match `import "xxx"`
	r, _ = regexp.Compile("\\r?\\nimport([\\s\\t]+)[\"`]([^\\r\\n]+)[\"`]([^\\r\\n]{0,1000})")
	res := r.FindAllStringSubmatch(txt, 100)
	for _, v := range res {
		if len(v) >= 2 {
			categorize(v[2], "", v[3])
		}
		if v[2] != "C" {
			txt = strings.Replace(txt, v[0], "", 1)
		}
	}
	// match `import xxx "xxx"` with alias
	r, _ = regexp.Compile("\\r?\\nimport([\\s\\t]+)(\\w+)([\\s\\t]+)[\"`]([^\\r\\n]+)[\"`]([^\\r\\n]{0,1000})")
	res = r.FindAllStringSubmatch(txt, 100)

	for _, v := range res {
		if len(v) >= 5 {
			categorize(v[4], v[2], "")
			txt = strings.Replace(txt, v[0], "", 1)
		}
	}
	// match `import ()`
	r, _ = regexp.Compile("\\r?\\nimport([\\s\\t\\r\\n]+\\()([^\\)]+)(\\))")
	res = r.FindAllStringSubmatch(txt, 100)
	for _, v := range res {
		if len(v) >= 3 {
			ss := strings.Split(v[2], separator)
			for _, val := range ss {
				val = strings.TrimSpace(strings.ReplaceAll(val, "\t", " "))
				if val != "" {
					if val[0] == '"' && val[len(val)-1] == '"' && (!strings.ContainsRune(val, ' ') && !strings.Contains(val, "\t")) {
						categorize(strings.Trim(val, `"`), "", "")
					} else if val[0] == '"' && strings.Contains(val, "//") || strings.Contains(val, "/*") {
						tmpPack := strings.SplitN(val, " ", 1)
						if len(tmpPack) > 1 {
							categorize(strings.Trim(tmpPack[0], `"`), "", strings.Trim(tmpPack[1], `"`))
						}
					} else {
						if val[:2] == "//" {
							categorize("", "", val)
						} else {
							tmpPack := strings.SplitN(val, " ", 3)
							if len(tmpPack) == 2 {
								categorize(strings.Trim(tmpPack[1], `"`), tmpPack[0], "")
							} else if len(tmpPack) == 3 {
								categorize(strings.Trim(tmpPack[1], `"`), tmpPack[0], tmpPack[2])
							}
						}
					}
				}
			}
			txt = strings.Replace(txt, v[0], "", 1)
		}
	}
	for strings.Contains(txt, separator+separator+separator) {
		txt = strings.Replace(txt, separator+separator+separator, separator+separator, -1)
	}
	write(file, txt)
}

func write(file, txt string) {
	pattern := "^package (\\w+)"
	if impC != "" {
		pattern = `\r?\nimport[\s\t]+"C"[^\r\n]{0,1000}`
	}
	r, _ := regexp.Compile(pattern)
	res := r.FindAllStringSubmatch(txt, 100)
	if len(res) > 0 && len(res[0]) >= 1 {
		newStr := res[0][0] + separator + separator
		if impC != "" {
			newStr = separator + impC + separator + separator
		}
		std, prj, ext := impBuilder()
		if len(std)+len(prj)+len(ext) == 1 {
			newStr += `import `
			if len(std) == 1 {
				newStr += strings.TrimSpace(std[0])
			} else if len(prj) == 1 {
				newStr += strings.TrimSpace(prj[0])
			} else {
				newStr += strings.TrimSpace(ext[0])
			}
			for _, v := range impAno {
				newStr += separator + "\t" + v
			}
		} else if len(std) > 0 || len(prj) > 0 || len(ext) > 0 {
			newStr += "import ("
			if len(std) > 0 {
				for _, v := range std {
					newStr += v
				}
				if len(prj) > 0 || len(ext) > 0 {
					newStr += separator
				}
			}

			if len(prj) > 0 {
				for _, v := range prj {
					newStr += v
				}
				if len(ext) > 0 {
					newStr += separator
				}
			}

			if len(ext) > 0 {
				for _, v := range ext {
					newStr += v
				}
			}
			if len(impAno) > 0 {
				newStr += separator
				for _, v := range impAno {
					newStr += separator + v
				}
			}

			newStr += separator + ")"
		}
		txt = strings.Replace(txt, res[0][0], newStr, 1)
	}
	for strings.Contains(txt, separator+separator+separator) {
		txt = strings.Replace(txt, separator+separator+separator, separator+separator, -1)
	}
	txt = strings.TrimSpace(txt) + separator
	//fmt.Println("_______________________\n", file, "\n", txt)
	ioutil.WriteFile(file, []byte(txt), os.ModePerm)
	reset()
}

func impBuilder() ([]string, []string, []string) {
	var (
		tmp []string
		std []string
		prj []string
		ext []string
	)
	// standard packages
	for name := range impStd {
		tmp = append(tmp, name)
	}
	sort.Strings(tmp)
	for _, name := range tmp {
		var temp []string
		for _, v := range impStd[name] {
			var isSet bool
			anno := ""
			if v.Annotation != "" {
				anno = "\t" + v.Annotation
			}
			if v.Name == "" {
				temp = append(temp, separator+"\t"+v.Annotation)
			} else if v.Alias == "" {
				if !isSet {
					temp = append(temp, separator+"\t\""+name+`"`+anno)
					isSet = true
				}
			} else {
				temp = append(temp, separator+"\t"+v.Alias+` "`+name+`"`+anno)
			}
		}
		sort.Strings(temp)
		std = append(std, temp...)
	}
	//fmt.Println(std)
	tmp = nil
	// project packages
	for name := range impPrj {
		tmp = append(tmp, name)
	}
	sort.Strings(tmp)
	for _, name := range tmp {
		var temp []string
		for _, v := range impPrj[name] {
			var isSet bool
			anno := ""
			if v.Annotation != "" {
				anno = "\t" + v.Annotation
			}
			if v.Alias == "" {
				if !isSet {
					temp = append(temp, separator+"\t"+`"`+name+`"`+anno)
					isSet = true
				}
			} else {
				//fmt.Printf("--------%#v, %d\n", v.Alias, len(v.Alias))
				temp = append(temp, separator+"\t"+v.Alias+` "`+name+`"`+anno)
			}
		}
		sort.Strings(temp)
		prj = append(prj, temp...)
	}
	//fmt.Println(prj)
	tmp = nil
	// extra packages
	for name := range impExt {
		tmp = append(tmp, name)
	}
	sort.Strings(tmp)
	for _, name := range tmp {
		var temp []string
		for _, v := range impExt[name] {
			var isSet bool
			anno := ""
			if v.Annotation != "" {
				anno = "\t" + v.Annotation
			}
			if v.Alias == "" {
				if !isSet {
					temp = append(temp, separator+"\t\""+name+`"`+anno)
					isSet = true
				}
			} else {
				temp = append(temp, separator+"\t"+v.Alias+` "`+name+`"`+anno)
			}
		}
		sort.Strings(temp)
		ext = append(ext, temp...)
	}
	return std, prj, ext
}

func reset() {
	impC = ""
	impStd = map[string][]Pack{}
	impPrj = map[string][]Pack{}
	impExt = map[string][]Pack{}
	impAno = []string{}
}

func execGofmt(dirPath string) {
	var stderr bytes.Buffer
	cmd := exec.Command("gofmt", "-s", "-w", dirPath)
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalln("[gofmt error]: ", err, " , [msg]: ", stderr.String())
	}
}

func categorize(packName, alias, annotation string) {
	if annotation != "" {
		annotation = strings.TrimSpace(annotation)
		for strings.Contains(annotation, "  ") {
			annotation = strings.Replace(annotation, "  ", " ", -1)
		}
		for strings.Contains(annotation, "\t\t") {
			annotation = strings.Replace(annotation, "\t\t", "\t", -1)
		}
		annotation = "\t" + annotation
	}
	if packName == "C" {
		impC = `import "C"` + annotation
	} else {
		if packName == "" {
			impAno = append(impAno, annotation)
		} else {
			ss := strings.Split(packName, "/")
			if _, ok := stdLibs[ss[0]]; ok {
				impStd[packName] = append(impStd[packName], Pack{packName, alias, annotation})
			} else if strings.Contains(ss[0], ".") {
				impExt[packName] = append(impExt[packName], Pack{packName, alias, annotation})
			} else {
				impPrj[packName] = append(impPrj[packName], Pack{packName, alias, annotation})
			}
		}
	}
}

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ChrisTrenkamp/goxpath"
	"github.com/ChrisTrenkamp/goxpath/tree"
	"github.com/ChrisTrenkamp/goxpath/tree/xmltree"
)

type namespace map[string]string

func (n *namespace) String() string {
	return fmt.Sprint(*n)
}

func (n *namespace) Set(value string) error {
	nsMap := strings.Split(value, "=")
	if len(nsMap) != 2 {
		nsErr = fmt.Errorf("Invalid namespace mapping: %s\n", value)
	}
	(*n)[nsMap[0]] = nsMap[1]
	return nil
}

type variables map[string]string

func (v *variables) String() string {
	return fmt.Sprint(*v)
}

func (v *variables) Set(value string) error {
	varMap := strings.Split(value, "=")
	if len(varMap) != 2 {
		nsErr = fmt.Errorf("Invalid variable mapping: %s\n", value)
	}
	(*v)[varMap[0]] = varMap[1]
	return nil
}

var rec bool
var value bool
var ns = make(namespace)
var vars = make(variables)
var nsErr error
var unstrict bool
var noFileName bool
var args = []string{}
var stdin io.Reader = os.Stdin
var stdout io.ReadWriter = os.Stdout
var stderr io.ReadWriter = os.Stderr

var retCode = 0

func main() {
	exec()
	os.Exit(retCode)
}

func exec() {
	flag.BoolVar(&rec, "r", false, "Recursive")
	flag.BoolVar(&value, "v", false, "Output the string value of the XPath result")
	flag.Var(&ns, "ns", "Namespace mappings. e.g. -ns myns=http://example.com")
	flag.Var(&vars, "var", "Variables mappings. e.g. -var myvar=myvalue")
	flag.BoolVar(&unstrict, "u", false, "Turns off strict XML validation")
	flag.BoolVar(&noFileName, "h", false, "Suppress filename prefixes.")
	flag.Parse()
	args = flag.Args()

	if nsErr != nil {
		fmt.Fprintf(stderr, nsErr.Error())
		retCode = 1
		return
	}

	if len(args) < 1 {
		fmt.Fprintf(stderr, "Specify an XPath expression with one or more files, or pipe the XML from stdin.  Run 'goxpath --help' for more information.\n")
		retCode = 1
		return
	}

	xp, err := goxpath.Parse(args[0])

	if err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		retCode = 1
		return
	}

	if len(args) == 1 {
		ret, err := runXPath(xp, stdin, ns, value)
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err.Error())
			retCode = 1
		}

		printResult(ret, "")
	}

	for i := 1; i < len(args); i++ {
		procPath(args[i], xp, ns, value)
	}
}

func procPath(path string, x goxpath.XPathExec, ns namespace, value bool) {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(stderr, "Could not open file: %s\n", path)
		retCode = 1
		return
	}

	if fi.IsDir() {
		procDir(path, x, ns, value)
		return
	}

	data, _ := ioutil.ReadFile(path)
	ret, err := runXPath(x, bytes.NewBuffer(data), ns, value)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %s\n", path, err.Error())
		retCode = 1
	}

	printResult(ret, path)
}

func printResult(ret []string, path string) {
	for _, j := range ret {
		if (len(flag.Args()) > 2 || rec) && !noFileName {
			fmt.Fprintf(stdout, "%s:", path)
		}

		fmt.Fprintf(stdout, "%s\n", j)
	}
}

func procDir(path string, x goxpath.XPathExec, ns namespace, value bool) {
	if !rec {
		fmt.Fprintf(stderr, "%s: Is a directory\n", path)
		retCode = 1
		return
	}

	list, _ := ioutil.ReadDir(path)

	for _, i := range list {
		procPath(filepath.Join(path, i.Name()), x, ns, value)
	}
}

func runXPath(x goxpath.XPathExec, r io.Reader, ns namespace, value bool) ([]string, error) {
	t, err := xmltree.ParseXML(r, func(o *xmltree.ParseOptions) {
		o.Strict = !unstrict
	})

	if err != nil {
		return nil, err
	}

	res, err := x.Exec(t, func(o *goxpath.Opts) {
		o.NS = ns
		for k, v := range vars {
			o.Vars[k] = tree.String(v)
		}
	})

	if err != nil {
		return nil, err
	}

	var ret []string

	if nodes, ok := res.(tree.NodeSet); ok && !value {
		ret = make([]string, len(nodes))
		for i, v := range nodes {
			ret[i], _ = goxpath.MarshalStr(v)
			ret[i] = strings.Replace(ret[i], "\n", "&#10;", -1)
		}
	} else {
		str := res.String()
		if str != "" {
			ret = strings.Split(str, "\n")
		}
	}

	return ret, nil
}

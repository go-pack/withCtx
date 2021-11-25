package main

import (
	"flag"
	"fmt"
	"go/token"
	"os"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
)

var (
	isOverWrite  = flag.Bool("m", false, "是否覆盖")
	output       = flag.String("o", "/tmp/test.go", "输出名称")
	isExtend     = flag.Bool("x", false, "是否扩展新方法")
	isAppend     = flag.Bool("a", false, "追加ctx参数")
	isProxy      = flag.Bool("p", false, "代理追加ctx参数")
	extendFunLog = make(map[string]bool, 3)
	proxyFunLog  = make(map[string]bool, 3)
)

//go:generate go version
func main() {

	flag.Parse()
	if len(os.Args) == 1 {
		flag.Usage()
		return
	}
	// *isProxy = true
	// *isOverWrite = true
	wd, _ := os.Getwd()
	file := os.Getenv("GOFILE")
	pack := os.Getenv("GOPACKAGE")
	path := wd + string(os.PathSeparator) + file

	// path = "/Users/chen/IdeaProjects/smm-go/internal/services/indexService.go"

	fmt.Printf("wd %s file %s pack %s path %s \r\n", wd, file, pack, path)

	// Create the AST by parsing src.
	fset := token.NewFileSet() // positions are relative to fset
	f, err := decorator.ParseFile(fset, path, nil, 0)
	if err != nil {
		panic(err)
	}
	var withCtx = "WithCtx"

	// Inspect the AST and print all identifiers and literals.
	dst.Inspect(f, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.FuncDecl:
			if strings.HasSuffix(x.Name.Name, withCtx) {
				fnName := strings.TrimRight(x.Name.Name, withCtx)
				extendFunLog[fnName] = true
				return true
			}
			fnName := x.Name.Name
			proxyFunLog[fnName] = true
		}
		return true
	})
	// 添加ctx 参数
	dst.Inspect(f, func(n dst.Node) bool {
		switch x := n.(type) {
		case *dst.GenDecl:
			if x.Tok == token.IMPORT {
				hasCtx := false
				for _, v := range x.Specs {
					if v.(*dst.ImportSpec).Path.Value == "\"context\"" {
						hasCtx = true
					}
				}
				if !hasCtx && x.Specs != nil && len(x.Specs) > 0 {
					cloned := dst.Clone(x.Specs[0]).(*dst.ImportSpec)
					cloned.Path.Value = "\"context\""
					x.Specs = append(x.Specs, cloned)
				}
			}
		case *dst.FuncDecl:
			if x.Recv != nil {
				if *isExtend {
					createExtendMethod(x, withCtx, f)

				} else if *isAppend {
					b, done := appendMethodArg(x)
					if done {
						return b
					}
				} else if *isProxy {
					leName := strings.ToLower(x.Name.Name[:1]) + x.Name.Name[1:]
					upName := strings.ToUpper(x.Name.Name[:1]) + x.Name.Name[1:]
					_, okLetter := proxyFunLog[leName]
					_, okUpper := proxyFunLog[upName]
					if okLetter && okUpper {
						return true
					}
					appendMethodArg(x)
					createProxyMethod(x, f)
				}
			}
		}
		return true
	})

	if *isOverWrite {
		*output = path
	}
	ret, _ := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE, 0666)
	if err := decorator.Fprint(ret, f); err != nil {
		panic(err)
	}
	// dst.Print(f)

}

func createExtendMethod(x *dst.FuncDecl, withCtx string, f *dst.File) (bool, bool) {
	_, ok := extendFunLog[x.Name.Name]
	if ok {
		return true, true
	}
	var ctxField = &dst.Field{
		Names: []*dst.Ident{
			{
				Name: "ctx",
				Obj:  &dst.Object{Kind: dst.ObjKind(token.VAR), Name: "ctx"},
			},
		},
		Type: &dst.SelectorExpr{
			X: &dst.Ident{Name: "context"}, Sel: &dst.Ident{Name: "Context"},
		},
	}
	funcDecl := dst.Clone(x).(*dst.FuncDecl)
	extendFunLog[x.Name.Name] = true
	// var  = strings.ToLower(funcDecl.Name.Name[:1]) + funcDecl.Name.Name[1:]

	var oldName = funcDecl.Name.Name
	var recName = x.Recv.List[0].Names[0].Name
	funcDecl.Name.Name = funcDecl.Name.Name + withCtx
	if len(funcDecl.Type.Params.List) == 0 || funcDecl.Type.Params.List == nil {
		funcDecl.Type.Params.List = append(make([]*dst.Field, 0), ctxField)
	} else {
		_, ok := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr)
		if ok {
			ctxPackage := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr).X.(*dst.Ident).Name
			ctxPackageLib := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr).Sel.Name
			if ctxPackage == "context" && ctxPackageLib == "Context" {
				return false, true
			}
		}
		if funcDecl.Type.Params.List[0].Names[0].Name != "ctx" {
			var fieds = make([]*dst.Field, 0)
			fieds = append(fieds, ctxField)
			for _, v := range funcDecl.Type.Params.List {
				var vv = v
				fieds = append(fieds, vv)
			}
			funcDecl.Type.Params.List = fieds
		}
	}
	var args = make([]dst.Expr, 0)
	for _, v := range x.Type.Params.List {
		args = append(args, &dst.Ident{
			Name: v.Names[0].Name,
		})
	}
	var results = make([]*dst.CallExpr, 0)
	results = append(results, &dst.CallExpr{
		Fun: &dst.SelectorExpr{
			X:   &dst.Ident{Name: recName},
			Sel: &dst.Ident{Name: oldName},
		},
		Args: args,
	})
	var resultsExpr = make([]dst.Expr, 0)
	for _, v := range results {
		resultsExpr = append(resultsExpr, v)
	}

	var bodyList = make([]dst.Stmt, 0)
	bodyList = append(bodyList, &dst.ReturnStmt{
		Results: resultsExpr,
	})
	funcDecl.Body.List = bodyList
	f.Decls = append(f.Decls, funcDecl)
	return false, false
}
func appendMethodArg(x *dst.FuncDecl) (bool, bool) {
	_, ok := extendFunLog[x.Name.Name]
	if ok {
		return true, true
	}
	var ctxField = &dst.Field{
		Names: []*dst.Ident{
			{
				Name: "ctx",
				Obj:  &dst.Object{Kind: dst.ObjKind(token.VAR), Name: "ctx"},
			},
		},
		Type: &dst.SelectorExpr{
			X: &dst.Ident{Name: "context"}, Sel: &dst.Ident{Name: "Context"},
		},
	}
	funcDecl := x
	extendFunLog[x.Name.Name] = true
	if len(funcDecl.Type.Params.List) == 0 || funcDecl.Type.Params.List == nil {
		funcDecl.Type.Params.List = append(make([]*dst.Field, 0), ctxField)
	} else {
		_, ok := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr)
		if ok {
			ctxPackage := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr).X.(*dst.Ident).Name
			ctxPackageLib := funcDecl.Type.Params.List[0].Type.(*dst.SelectorExpr).Sel.Name
			if ctxPackage == "context" && ctxPackageLib == "Context" {
				return false, true
			}
		}
		if funcDecl.Type.Params.List[0].Names[0].Name != "ctx" {
			var fieds = make([]*dst.Field, 0)
			fieds = append(fieds, ctxField)
			for _, v := range funcDecl.Type.Params.List {
				var vv = v
				fieds = append(fieds, vv)
			}
			funcDecl.Type.Params.List = fieds
		}
	}
	return false, false
}

func createProxyMethod(x *dst.FuncDecl, f *dst.File) (bool, bool) {
	leName := strings.ToLower(x.Name.Name[:1]) + x.Name.Name[1:]
	upName := strings.ToUpper(x.Name.Name[:1]) + x.Name.Name[1:]
	_, okLetter := proxyFunLog[leName]
	_, okUpper := proxyFunLog[upName]
	if okLetter && okUpper {
		return false, false
	}
	//如果是私有方法不处理
	if x.Name.Name[:1] == strings.ToLower(x.Name.Name[:1]) {
		return false, false
	}
	funcDecl := dst.Clone(x).(*dst.FuncDecl)
	extendFunLog[x.Name.Name] = true
	x.Name.Name = leName
	var recName = x.Recv.List[0].Names[0].Name
	funcDecl.Name.Name = upName

	var args = make([]dst.Expr, 0)
	for _, v := range x.Type.Params.List {
		args = append(args, &dst.Ident{
			Name: v.Names[0].Name,
		})
	}
	var results = make([]*dst.CallExpr, 0)
	results = append(results, &dst.CallExpr{
		Fun: &dst.SelectorExpr{
			X:   &dst.Ident{Name: recName},
			Sel: &dst.Ident{Name: leName},
		},
		Args: args,
	})
	var resultsExpr = make([]dst.Expr, 0)
	for _, v := range results {
		resultsExpr = append(resultsExpr, v)
	}

	var bodyList = make([]dst.Stmt, 0)
	bodyList = append(bodyList, &dst.ReturnStmt{
		Results: resultsExpr,
	})
	funcDecl.Body.List = bodyList
	f.Decls = append(f.Decls, funcDecl)
	return false, false
}

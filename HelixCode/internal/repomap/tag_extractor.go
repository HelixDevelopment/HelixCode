package repomap

import (
	"bufio"
	"os"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// SymbolType represents the type of a code symbol
type SymbolType string

const (
	SymbolTypeFunction  SymbolType = "function"
	SymbolTypeMethod    SymbolType = "method"
	SymbolTypeClass     SymbolType = "class"
	SymbolTypeInterface SymbolType = "interface"
	SymbolTypeStruct    SymbolType = "struct"
	SymbolTypeEnum      SymbolType = "enum"
	SymbolTypeTrait     SymbolType = "trait"
	SymbolTypeModule    SymbolType = "module"
	SymbolTypeVariable  SymbolType = "variable"
	SymbolTypeConstant  SymbolType = "constant"
	SymbolTypeImport    SymbolType = "import"
	SymbolTypeExport    SymbolType = "export"
)

// Symbol represents a code symbol (function, class, method, etc.)
type Symbol struct {
	Name      string
	Type      SymbolType
	FilePath  string
	LineStart int
	LineEnd   int
	Signature string
	Docstring string
	Parent    string // For methods, the class/struct they belong to
}

// TagExtractor extracts symbols from syntax trees
type TagExtractor struct {
	language string
}

// NewTagExtractor creates a new tag extractor for a specific language
func NewTagExtractor(language string) *TagExtractor {
	return &TagExtractor{
		language: language,
	}
}

// Extract extracts all symbols from a syntax tree
func (te *TagExtractor) Extract(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	switch te.language {
	case "go":
		symbols = te.extractGoSymbols(tree, filePath)
	case "python":
		symbols = te.extractPythonSymbols(tree, filePath)
	case "javascript", "typescript":
		symbols = te.extractJavaScriptSymbols(tree, filePath)
	case "java":
		symbols = te.extractJavaSymbols(tree, filePath)
	case "c", "cpp":
		symbols = te.extractCppSymbols(tree, filePath)
	case "rust":
		symbols = te.extractRustSymbols(tree, filePath)
	case "ruby":
		symbols = te.extractRubySymbols(tree, filePath)
	}

	return symbols
}

// extractGoSymbols extracts symbols from Go code
func (te *TagExtractor) extractGoSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	// Extract functions
	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			symbol := te.extractGoFunction(node, filePath)
			symbols = append(symbols, symbol)
		case "method_declaration":
			symbol := te.extractGoMethod(node, filePath)
			symbols = append(symbols, symbol)
		case "type_declaration":
			typeSymbols := te.extractGoType(node, filePath)
			symbols = append(symbols, typeSymbols...)
		}
	})

	return symbols
}

// extractGoFunction extracts a Go function symbol
func (te *TagExtractor) extractGoFunction(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeFunction,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractGoMethod extracts a Go method symbol
func (te *TagExtractor) extractGoMethod(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	receiverNode := node.ChildByFieldName("receiver")

	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	receiver := ""
	if receiverNode != nil {
		receiver = te.getNodeText(filePath, receiverNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeMethod,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
		Parent:    receiver,
	}
}

// extractGoType extracts Go type symbols (struct, interface)
func (te *TagExtractor) extractGoType(node *sitter.Node, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "type_spec" {
			continue
		}

		nameNode := child.ChildByFieldName("name")
		typeNode := child.ChildByFieldName("type")

		if nameNode == nil || typeNode == nil {
			continue
		}

		name := te.getNodeText(filePath, nameNode)
		symbolType := SymbolTypeStruct

		if typeNode.Type() == "interface_type" {
			symbolType = SymbolTypeInterface
		}

		symbols = append(symbols, Symbol{
			Name:      name,
			Type:      symbolType,
			FilePath:  filePath,
			LineStart: int(child.StartPoint().Row) + 1,
			LineEnd:   int(child.EndPoint().Row) + 1,
			Signature: te.getNodeText(filePath, child),
			Docstring: te.extractDocstring(filePath, int(child.StartPoint().Row)),
		})
	}

	return symbols
}

// extractPythonSymbols extracts symbols from Python code
func (te *TagExtractor) extractPythonSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "function_definition":
			symbol := te.extractPythonFunction(node, filePath, "")
			symbols = append(symbols, symbol)
		case "class_definition":
			classSymbols := te.extractPythonClass(node, filePath)
			symbols = append(symbols, classSymbols...)
		}
	})

	return symbols
}

// extractPythonFunction extracts a Python function/method symbol
func (te *TagExtractor) extractPythonFunction(node *sitter.Node, filePath string, parent string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	symbolType := SymbolTypeFunction
	if parent != "" {
		symbolType = SymbolTypeMethod
	}

	return Symbol{
		Name:      name,
		Type:      symbolType,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractPythonDocstring(node, filePath),
		Parent:    parent,
	}
}

// extractPythonClass extracts a Python class and its methods
func (te *TagExtractor) extractPythonClass(node *sitter.Node, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	nameNode := node.ChildByFieldName("name")
	className := ""
	if nameNode != nil {
		className = te.getNodeText(filePath, nameNode)
	}

	// Add class symbol
	symbols = append(symbols, Symbol{
		Name:      className,
		Type:      SymbolTypeClass,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractPythonDocstring(node, filePath),
	})

	// Extract methods
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		te.walkTree(bodyNode, func(child *sitter.Node) {
			if child.Type() == "function_definition" {
				method := te.extractPythonFunction(child, filePath, className)
				symbols = append(symbols, method)
			}
		})
	}

	return symbols
}

// extractJavaScriptSymbols extracts symbols from JavaScript/TypeScript code
func (te *TagExtractor) extractJavaScriptSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "function_declaration":
			symbol := te.extractJSFunction(node, filePath)
			symbols = append(symbols, symbol)
		case "class_declaration":
			classSymbols := te.extractJSClass(node, filePath)
			symbols = append(symbols, classSymbols...)
		case "variable_declarator":
			if te.isArrowFunction(node) {
				symbol := te.extractJSArrowFunction(node, filePath)
				symbols = append(symbols, symbol)
			}
		}
	})

	return symbols
}

// extractJSFunction extracts a JavaScript function symbol
func (te *TagExtractor) extractJSFunction(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeFunction,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJSDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractJSClass extracts a JavaScript class and its methods
func (te *TagExtractor) extractJSClass(node *sitter.Node, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	nameNode := node.ChildByFieldName("name")
	className := ""
	if nameNode != nil {
		className = te.getNodeText(filePath, nameNode)
	}

	symbols = append(symbols, Symbol{
		Name:      className,
		Type:      SymbolTypeClass,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJSDocstring(filePath, int(node.StartPoint().Row)),
	})

	// Extract methods
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		te.walkTree(bodyNode, func(child *sitter.Node) {
			if child.Type() == "method_definition" {
				method := te.extractJSMethod(child, filePath, className)
				symbols = append(symbols, method)
			}
		})
	}

	return symbols
}

// extractJSMethod extracts a JavaScript method symbol
func (te *TagExtractor) extractJSMethod(node *sitter.Node, filePath string, parent string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeMethod,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJSDocstring(filePath, int(node.StartPoint().Row)),
		Parent:    parent,
	}
}

// extractJSArrowFunction extracts an arrow function symbol
func (te *TagExtractor) extractJSArrowFunction(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeFunction,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJSDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// isArrowFunction checks if a variable declarator contains an arrow function
func (te *TagExtractor) isArrowFunction(node *sitter.Node) bool {
	valueNode := node.ChildByFieldName("value")
	return valueNode != nil && valueNode.Type() == "arrow_function"
}

// extractJavaSymbols extracts symbols from Java code
func (te *TagExtractor) extractJavaSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "class_declaration":
			classSymbols := te.extractJavaClass(node, filePath)
			symbols = append(symbols, classSymbols...)
		case "interface_declaration":
			symbol := te.extractJavaInterface(node, filePath)
			symbols = append(symbols, symbol)
		}
	})

	return symbols
}

// extractJavaClass extracts a Java class and its methods
func (te *TagExtractor) extractJavaClass(node *sitter.Node, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	nameNode := node.ChildByFieldName("name")
	className := ""
	if nameNode != nil {
		className = te.getNodeText(filePath, nameNode)
	}

	symbols = append(symbols, Symbol{
		Name:      className,
		Type:      SymbolTypeClass,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJavaDocstring(filePath, int(node.StartPoint().Row)),
	})

	// Extract methods
	bodyNode := node.ChildByFieldName("body")
	if bodyNode != nil {
		te.walkTree(bodyNode, func(child *sitter.Node) {
			if child.Type() == "method_declaration" {
				method := te.extractJavaMethod(child, filePath, className)
				symbols = append(symbols, method)
			}
		})
	}

	return symbols
}

// extractJavaMethod extracts a Java method symbol
func (te *TagExtractor) extractJavaMethod(node *sitter.Node, filePath string, parent string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeMethod,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJavaDocstring(filePath, int(node.StartPoint().Row)),
		Parent:    parent,
	}
}

// extractJavaInterface extracts a Java interface symbol
func (te *TagExtractor) extractJavaInterface(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeInterface,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractJavaDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractCppSymbols extracts symbols from C/C++ code
func (te *TagExtractor) extractCppSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "function_definition":
			symbol := te.extractCppFunction(node, filePath)
			symbols = append(symbols, symbol)
		case "class_specifier":
			symbol := te.extractCppClass(node, filePath)
			symbols = append(symbols, symbol)
		case "struct_specifier":
			symbol := te.extractCppStruct(node, filePath)
			symbols = append(symbols, symbol)
		}
	})

	return symbols
}

// extractCppFunction extracts a C++ function symbol
func (te *TagExtractor) extractCppFunction(node *sitter.Node, filePath string) Symbol {
	declaratorNode := node.ChildByFieldName("declarator")
	name := ""
	if declaratorNode != nil {
		// Try to find identifier in declarator
		name = te.extractCppFunctionName(declaratorNode, filePath)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeFunction,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractCppFunctionName extracts function name from declarator
func (te *TagExtractor) extractCppFunctionName(node *sitter.Node, filePath string) string {
	if node == nil {
		return ""
	}

	if node.Type() == "identifier" {
		return te.getNodeText(filePath, node)
	}

	// Recursively search for identifier
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if name := te.extractCppFunctionName(child, filePath); name != "" {
			return name
		}
	}

	return ""
}

// extractCppClass extracts a C++ class symbol
func (te *TagExtractor) extractCppClass(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeClass,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractCppStruct extracts a C++ struct symbol
func (te *TagExtractor) extractCppStruct(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeStruct,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractRustSymbols extracts symbols from Rust code
func (te *TagExtractor) extractRustSymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "function_item":
			symbol := te.extractRustFunction(node, filePath)
			symbols = append(symbols, symbol)
		case "struct_item":
			symbol := te.extractRustStruct(node, filePath)
			symbols = append(symbols, symbol)
		case "enum_item":
			symbol := te.extractRustEnum(node, filePath)
			symbols = append(symbols, symbol)
		case "trait_item":
			symbol := te.extractRustTrait(node, filePath)
			symbols = append(symbols, symbol)
		}
	})

	return symbols
}

// extractRustFunction extracts a Rust function symbol
func (te *TagExtractor) extractRustFunction(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeFunction,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractRustStruct extracts a Rust struct symbol
func (te *TagExtractor) extractRustStruct(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeStruct,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractRustEnum extracts a Rust enum symbol
func (te *TagExtractor) extractRustEnum(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeEnum,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractRustTrait extracts a Rust trait symbol
func (te *TagExtractor) extractRustTrait(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeTrait,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// extractRubySymbols extracts symbols from Ruby code
func (te *TagExtractor) extractRubySymbols(tree *sitter.Tree, filePath string) []Symbol {
	symbols := make([]Symbol, 0)
	rootNode := tree.RootNode()

	te.walkTree(rootNode, func(node *sitter.Node) {
		switch node.Type() {
		case "method":
			symbol := te.extractRubyMethod(node, filePath, "")
			symbols = append(symbols, symbol)
		case "class":
			classSymbols := te.extractRubyClass(node, filePath)
			symbols = append(symbols, classSymbols...)
		case "module":
			symbol := te.extractRubyModule(node, filePath)
			symbols = append(symbols, symbol)
		}
	})

	return symbols
}

// extractRubyMethod extracts a Ruby method symbol
func (te *TagExtractor) extractRubyMethod(node *sitter.Node, filePath string, parent string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	symbolType := SymbolTypeFunction
	if parent != "" {
		symbolType = SymbolTypeMethod
	}

	return Symbol{
		Name:      name,
		Type:      symbolType,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
		Parent:    parent,
	}
}

// extractRubyClass extracts a Ruby class and its methods
func (te *TagExtractor) extractRubyClass(node *sitter.Node, filePath string) []Symbol {
	symbols := make([]Symbol, 0)

	nameNode := node.ChildByFieldName("name")
	className := ""
	if nameNode != nil {
		className = te.getNodeText(filePath, nameNode)
	}

	symbols = append(symbols, Symbol{
		Name:      className,
		Type:      SymbolTypeClass,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	})

	// Extract methods
	te.walkTree(node, func(child *sitter.Node) {
		if child.Type() == "method" {
			method := te.extractRubyMethod(child, filePath, className)
			symbols = append(symbols, method)
		}
	})

	return symbols
}

// extractRubyModule extracts a Ruby module symbol
func (te *TagExtractor) extractRubyModule(node *sitter.Node, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = te.getNodeText(filePath, nameNode)
	}

	return Symbol{
		Name:      name,
		Type:      SymbolTypeModule,
		FilePath:  filePath,
		LineStart: int(node.StartPoint().Row) + 1,
		LineEnd:   int(node.EndPoint().Row) + 1,
		Signature: te.getNodeText(filePath, node),
		Docstring: te.extractDocstring(filePath, int(node.StartPoint().Row)),
	}
}

// Helper methods

// walkTree walks the syntax tree and calls the visitor function for each node
func (te *TagExtractor) walkTree(node *sitter.Node, visitor func(*sitter.Node)) {
	if node == nil {
		return
	}

	visitor(node)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		te.walkTree(child, visitor)
	}
}

// getNodeText extracts the text content of a node
func (te *TagExtractor) getNodeText(filePath string, node *sitter.Node) string {
	if node == nil {
		return ""
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	start := node.StartByte()
	end := node.EndByte()

	if start >= uint32(len(content)) || end > uint32(len(content)) {
		return ""
	}

	return string(content[start:end])
}

// extractDocstring extracts documentation comment before a line
func (te *TagExtractor) extractDocstring(filePath string, line int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	docLines := make([]string, 0)

	for scanner.Scan() {
		lineNum++
		if lineNum >= line {
			break
		}

		text := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(text, "//") || strings.HasPrefix(text, "#") {
			docLines = append(docLines, text)
		} else if text != "" {
			docLines = make([]string, 0)
		}
	}

	return strings.Join(docLines, "\n")
}

// extractPythonDocstring extracts Python docstring
func (te *TagExtractor) extractPythonDocstring(node *sitter.Node, filePath string) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		return ""
	}

	// Look for first string expression (docstring)
	for i := 0; i < int(bodyNode.ChildCount()); i++ {
		child := bodyNode.Child(i)
		if child.Type() == "expression_statement" {
			exprChild := child.Child(0)
			if exprChild != nil && exprChild.Type() == "string" {
				return te.getNodeText(filePath, exprChild)
			}
		}
		break
	}

	return ""
}

// extractJSDocstring extracts JSDoc-style comments
func (te *TagExtractor) extractJSDocstring(filePath string, line int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	docLines := make([]string, 0)
	inJSDoc := false

	for scanner.Scan() {
		lineNum++
		if lineNum >= line {
			break
		}

		text := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(text, "/**") {
			inJSDoc = true
			docLines = make([]string, 0)
			docLines = append(docLines, text)
		} else if inJSDoc {
			docLines = append(docLines, text)
			if strings.Contains(text, "*/") {
				break
			}
		} else if strings.HasPrefix(text, "//") {
			docLines = append(docLines, text)
		} else if text != "" && !inJSDoc {
			docLines = make([]string, 0)
		}
	}

	return strings.Join(docLines, "\n")
}

// extractJavaDocstring extracts JavaDoc-style comments
func (te *TagExtractor) extractJavaDocstring(filePath string, line int) string {
	return te.extractJSDocstring(filePath, line) // Same format
}

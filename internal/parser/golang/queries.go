package golang

// queryFunctions matches top-level function declarations.
//
//	(function_declaration
//	  name: (identifier) @func.name
//	  parameters: (parameter_list) @func.params
//	  result: (_)? @func.result
//	  body: (block) @func.body) @func.def
var queryFunctions = []byte(`
(function_declaration
  name: (identifier) @func.name
  parameters: (parameter_list) @func.params
  body: (block) @func.body) @func.def
`)

// queryMethods matches method declarations (with a receiver).
var queryMethods = []byte(`
(method_declaration
  receiver: (parameter_list) @method.receiver
  name: (field_identifier) @method.name
  parameters: (parameter_list) @method.params
  body: (block) @method.body) @method.def
`)

// queryTypes matches named type declarations.
var queryTypes = []byte(`
(type_declaration
  (type_spec
    name: (type_identifier) @type.name
    type: (_) @type.body)) @type.def
`)

// queryConsts matches const declaration specs.
var queryConsts = []byte(`
(const_declaration
  (const_spec
    name: (identifier) @const.name)) @const.def
`)

// queryVars matches var declaration specs.
var queryVars = []byte(`
(var_declaration
  (var_spec
    name: (identifier) @var.name)) @var.def
`)

// queryDirectCalls matches direct function call expressions.
var queryDirectCalls = []byte(`
(call_expression
  function: (identifier) @call.func) @call.expr
`)

// querySelectorCalls matches selector call expressions (e.g. pkg.Func or obj.Method).
var querySelectorCalls = []byte(`
(call_expression
  function: (selector_expression
    operand: (_) @call.receiver
    field: (field_identifier) @call.func)) @call.expr
`)

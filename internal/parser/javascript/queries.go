package javascript

// queryFunctions matches top-level function declarations.
var queryFunctions = []byte(`
(function_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params
  body: (statement_block)) @func.def
`)

// queryClasses matches class declarations.
// JavaScript uses (identifier) for class names (TypeScript uses type_identifier).
var queryClasses = []byte(`
(class_declaration
  name: (identifier) @class.name
  body: (class_body)) @class.def
`)

// queryMethods matches method definitions inside class bodies.
var queryMethods = []byte(`
(method_definition
  name: (property_identifier) @method.name
  parameters: (formal_parameters) @method.params
  body: (statement_block)) @method.def
`)

// queryDirectCalls matches direct function call expressions.
var queryDirectCalls = []byte(`
(call_expression
  function: (identifier) @call.func) @call.expr
`)

// querySelectorCalls matches member call expressions (obj.method()).
var querySelectorCalls = []byte(`
(call_expression
  function: (member_expression
    object: (_) @call.receiver
    property: (property_identifier) @call.func)) @call.expr
`)

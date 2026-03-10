package typescript

// queryFunctions matches top-level function declarations.
var queryFunctions = []byte(`
(function_declaration
  name: (identifier) @func.name
  parameters: (formal_parameters) @func.params
  body: (statement_block)) @func.def
`)

// queryClasses matches class declarations.
var queryClasses = []byte(`
(class_declaration
  name: (type_identifier) @class.name
  body: (class_body)) @class.def
`)

// queryMethods matches method definitions inside class bodies.
var queryMethods = []byte(`
(method_definition
  name: (property_identifier) @method.name
  parameters: (formal_parameters) @method.params
  body: (statement_block)) @method.def
`)

// queryInterfaces matches interface declarations (TypeScript-specific).
var queryInterfaces = []byte(`
(interface_declaration
  name: (type_identifier) @iface.name
  body: (interface_body)) @iface.def
`)

// queryTypeAliases matches type alias declarations (TypeScript-specific).
var queryTypeAliases = []byte(`
(type_alias_declaration
  name: (type_identifier) @type.name) @type.def
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

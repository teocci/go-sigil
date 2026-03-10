package python

// queryFunctions matches all function definitions (module-level and class methods).
// The caller distinguishes functions from methods by checking the parent class.
var queryFunctions = []byte(`
(function_definition
  name: (identifier) @func.name
  parameters: (parameters) @func.params
  body: (block)) @func.def
`)

// queryClasses matches class definitions.
var queryClasses = []byte(`
(class_definition
  name: (identifier) @class.name
  body: (block)) @class.def
`)

// queryDirectCalls matches direct function call expressions.
var queryDirectCalls = []byte(`
(call
  function: (identifier) @call.func) @call.expr
`)

// queryAttributeCalls matches attribute/method call expressions (obj.method()).
var queryAttributeCalls = []byte(`
(call
  function: (attribute
    attribute: (identifier) @call.func)) @call.expr
`)

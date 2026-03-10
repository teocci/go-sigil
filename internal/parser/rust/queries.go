package rust

// queryFunctions matches top-level free functions.
var queryFunctions = []byte(`
(function_item
  name: (identifier) @fn.name
  parameters: (parameters) @fn.params
  body: (block) @fn.body) @fn.def
`)

// queryImplMethods matches methods inside impl blocks.
var queryImplMethods = []byte(`
(impl_item
  type: (_) @impl.type
  body: (declaration_list
    (function_item
      name: (identifier) @method.name
      parameters: (parameters) @method.params
      body: (block) @method.body) @method.def))
`)

// queryStructs matches struct declarations.
var queryStructs = []byte(`
(struct_item
  name: (type_identifier) @struct.name) @struct.def
`)

// queryTraits matches trait declarations.
var queryTraits = []byte(`
(trait_item
  name: (type_identifier) @trait.name) @trait.def
`)

// queryEnums matches enum declarations.
var queryEnums = []byte(`
(enum_item
  name: (type_identifier) @enum.name) @enum.def
`)

// queryTypeAliases matches type alias declarations.
var queryTypeAliases = []byte(`
(type_item
  name: (type_identifier) @type.name) @type.def
`)

// queryConsts matches const declarations.
var queryConsts = []byte(`
(const_item
  name: (identifier) @const.name) @const.def
`)

// queryDirectCalls matches direct function call expressions.
var queryDirectCalls = []byte(`
(call_expression
  function: (identifier) @call.func) @call.expr
`)

// queryScopedCalls matches scoped/method call expressions (Foo::bar or obj.method()).
var queryScopedCalls = []byte(`
(call_expression
  function: (scoped_identifier
    name: (identifier) @call.func)) @call.expr
`)

// queryMethodCalls matches method call expressions (obj.method()).
var queryMethodCalls = []byte(`
(method_call_expression
  method: (field_identifier) @call.func) @call.expr
`)

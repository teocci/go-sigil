package java

// queryClasses matches class declarations.
var queryClasses = []byte(`
(class_declaration
  name: (identifier) @class.name
  body: (class_body) @class.body) @class.def
`)

// queryInterfaces matches interface declarations.
var queryInterfaces = []byte(`
(interface_declaration
  name: (identifier) @interface.name) @interface.def
`)

// queryEnums matches enum declarations.
var queryEnums = []byte(`
(enum_declaration
  name: (identifier) @enum.name) @enum.def
`)

// queryMethods matches method declarations inside class/interface bodies.
var queryMethods = []byte(`
(method_declaration
  name: (identifier) @method.name
  parameters: (formal_parameters) @method.params
  body: (block) @method.body) @method.def
`)

// queryConstructors matches constructor declarations.
var queryConstructors = []byte(`
(constructor_declaration
  name: (identifier) @ctor.name
  parameters: (formal_parameters) @ctor.params
  body: (constructor_body) @ctor.body) @ctor.def
`)

// queryFields matches field declarations at class level.
var queryFields = []byte(`
(field_declaration
  declarator: (variable_declarator
    name: (identifier) @field.name)) @field.def
`)

// queryDirectCalls matches direct method invocations: method(args).
var queryDirectCalls = []byte(`
(method_invocation
  name: (identifier) @call.func
  arguments: (argument_list)) @call.expr
`)

// queryScopedCalls matches scoped method invocations: obj.method(args).
var queryScopedCalls = []byte(`
(method_invocation
  object: (_) @call.receiver
  name: (identifier) @call.func
  arguments: (argument_list)) @call.expr
`)

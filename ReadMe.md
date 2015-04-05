# GoJade

Jade template rendering library for Google Go programing language (golang). GoJade is inspired by the javascript template library http://jade-lang.com/

GoJade renders a jade file directly to HTML. 

all examples on http://jade-lang.com/ and http://jade-lang.com/reference/ is working except for one or two. See status for more details.

pull request welcomed.

Version: 0.012 Alpha

This library is under development and changes can be expected.

MIT License (MIT)

## Using GoJade

**Installing gojade**

```bash
go get github.com/zdebeer99/gojade 
```

**Importing into your project**

'''go
import (
  "github.com/zdebeer99/gojade"
)
'''

**Basic Usage**

```go
import (
  "github.com/zdebeer99/gojade"
  "fmt"
)

func main(){
  //GoJade can be declared globally. it will cache templates and keep config information for parsing.
  jade:=gojade.NewGoJade()
  jade.ViewPath = "./view"

  //RenderFile renders a jade file into html.
  fmt.Println(jade.RenderFile("index.jade", nil).String())
}

```

## Jade Examples

Jade Example:

```jade
doctype html
html(lang="en")
  head
    title= pageTitle
    script(type='text/javascript').
      if (foo) {
         bar(1 + 5)
      }
  body
    h1 Jade - node template engine
    #container.col
      if youAreUsingJade
        p You are amazing
      else
        p Get on it!
      p.
        Jade is a terse and simple
        templating language with a
        strong focus on performance
        and powerful features.
```

gojade supports math operators, operator precedence, boolean operators and string concatenation.

Examples:

```jade
// declare a variable called x
-var x = 5
p Do some maths with x = 5
p x * 2 = #{x*2}
p More Maths (x+3)*2
p=(x+3)*2
// some boolean operations
if x>=0 && x<10
  p x is smaller than ten
else 
  p x is equal or larger than 10
```

For more example's see the example folder included. The example folder contains a standard http example using gojade and a gin go web framework example using gojade instead of the standard go templates.


## Differences between jade and gojade

all examples on http://jade-lang.com/ and http://jade-lang.com/reference/ is working except for one or two. See status for more details.

**javascript**

Keep in mind that gojade does not run in a javascript environment, because of this server side javascript in gojade is not supported as in jade, a work around will be to register custom functions defined in go and calling these functions from your jade template.

Registering Custom Functions
```go
jade:=NewGoJade()
// Register a function called hello, that can be used in your jade template
jade.RegisterFunction("hello", func(name string){return "Hello"+name})
```

```jade
p= hello("Ben")
```

**variable names**

Variable names is case sensitive

Only public fields can be accessed from a struct data argument.

If a variable is not defined a warning will be Logged and nil is returned.

nil will eval to false


**doctype**

the doctype shortcut does not support custom doctypes.

**Boolean Attribute**

the Boolean Attribute
```jade
input(type='checkbox', checked=true && 'checked')
```

is not supported and no plans for support is currently in the pipeline, please log an issue if required.
use the ? conditional instead. Example:
```jade
input(type='checkbox', checked=true ? 'checked')
```


**Unbuffered Code**

Full javascript support for unbuffered code will not be supported as the template runs in go runtime.

Special cases are:

Defining a variable using var

```jade
- var x = 5
- var person = {name:"ben", age:5}
```

[Planned] Calling Unbuffered Functions defined in go.

```jade
- SomeGoFunction(5,64)
  p This Content will be passed to the last argument of SomeGoFunction()
  p This content is outside of SomeGoFunctions scope.
```

[may change] a unbuffered function must take either a string or a *TreeNode as the last argument. the content below the unbeffered function will be passed to the function. If the function takes a string the content will first be parsed and then passed to the function.

## Status

**Whats is out standing**

- filters and include with filters
- calling go functions from jade as unbuffered code.
- Tag Interpolation

gojade has not been cleaned up yet and some function names and parameters may change. The parser is currently pretty stable but evaluating the parse tree use panic instead of returning the error as a result. This will be changed to return the error rather than panicking, and there for some functions return arguments will change.

benchmarks and performance improvements has not been done yet.


## Useful functions

because gojade does not support javascript, the following will not work.

somestring.toUpperCase()
someArray.length

to assist with these scenarios you can register your own go functions.

Some Functions already included is:

* len(value)

* upper(string)

* lower(string)

* format(string,args...) - the same as fmt.SPrintf() in go

* more is planned, suggestions is welcomed.



golang jade go jade gojade 
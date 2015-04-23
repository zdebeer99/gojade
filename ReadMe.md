# GoJade

Jade template rendering library for the Go programing language (golang). GoJade is inspired by the javascript template library http://jade-lang.com/

GoJade renders a jade file directly to HTML.

all examples on http://jade-lang.com/ and http://jade-lang.com/reference/ is working except for inline tags and filters. See status for more details.

pull request welcomed.

MIT License (MIT)


## Using GoJade


**Installing gojade**


```bash
go get github.com/zdebeer99/gojade
```


**Importing into your project**

```go
import (
  "github.com/zdebeer99/gojade"
)
```


**Basic Usage**

```go
import (
  "github.com/zdebeer99/gojade"
  "fmt"
)

func main(){
  //GoJade can be declared globally. it will cache templates and keep config information for parsing.
  jade:=gojade.New()
  jade.ViewPath = "./view"

  //RenderFile renders a jade file into html.
  fmt.Println(jade.RenderFile("index.jade", nil).String())
}

```


## GoJade Examples

See the [Example Folder](https://github.com/zdebeer99/gojade/tree/master/example)
folder for examples. Currently the example folder includes;
* standard [net/http example](http://golang.org/pkg/net/http/) go web example.
* [gin](https://gin-gonic.github.io/gin/) web framework example.


gojade supports math operators, operator precedence, boolean operators and string concatenation.

```jade
-var x = 5
p Do some maths with x = 5
p x * 2 = #{x*2}
p More Maths (x+3)*2
p=(x+3)*2
//- boolean operations
if x>=0 && x<10
  p x is smaller than ten
else
  p x is equal or larger than 10
```


## Status


**What is out standing**

- Filters
- Tag Interpolation

gojade requires a clean up, some function names and parameters may change.

debug information can be improved.

benchmarks and performance improvements has not been done yet.


## Differences between jade and gojade


**javascript**

Keep in mind that gojade does not run in a javascript environment, because of this server side javascript in gojade is not supported as in jade, a work around will be to register custom functions defined in go and calling these functions from your jade template.

Registering Custom Functions.
```go
jade:=gojade.New()
// Register a function called hello, that can be used in your jade template
jade.RegisterFunction("hello", func(name string){return "Hello"+name})
```

Using the function in your jade.
```jade
p= hello("Ben")
```
methods defined on the model struct passed to the render function is also accessible from jade.

**Variables**

* Variable names is case sensitive.

* Only public fields and methods can be accessed from a struct passed to the render method.

* nil will eval to an empty string or false in a condition.

* calling a field or a method on a nil variable will throw an error.
    Example: lets say the object person is null then "person.Name" will throw an error. but just "person" will return a empty string.

**doctype**

The doctype shortcut does not support custom doctypes.


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
//Future Implementation, not done yet.
- SomeGoFunction(5,64)
  p This Content will be passed to the last argument of SomeGoFunction()
  p This content is outside of SomeGoFunctions scope.
```

## Useful functions

because gojade does not support javascript, the following will not work.

```javascript
somestring.toUpperCase()
someArray.length
```

in go strings does not have methods linked to the string. The Time type for example in go has methods map to it, show this will work.

```go
type Model struct{
  Time time.Time
}
```

```jade
p The Current date is
p= Time.Format("02 Jan 2006")
```

To assist with these scenarios gojade includes some built in functions and you can register your own go functions, any struct that has methods passed to the render function it's methods will also automatically be available in the Jade parser.

Built in Functions included is:

* len(value)
  Get the length of an array or string.

* upper(string)

* lower(string)

* format(string,args...) - the same as fmt.SPrintf() in go

* isnull(value) bool

* ifnull(value, replace) - returns replace if value is nil else it returns value

* more is planned, suggestions is welcomed.



golang jade go jade gojade

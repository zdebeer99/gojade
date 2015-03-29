# Go Jade

Version: 0.01 Alpha

This library is under developement and changes can be expected.

MIT License (MIT)

Go jade is inspired by the javascript template library http://jade-lang.com/

gojade renders a jade file directly to html, and does not compile to a gotemplate first.

While most of the Jade specifications is supported there is stil some jade features not yet fully supported, see the status section for information on this.

pull request and issues is welcomed.



## Differences between jade and gojade

**javascript**

Keep in mind that gojade does not run in a javascript enviroment, nor is it executed in a javascript
environment. Becuase of this javascript in gojade is not supported, a work around will be to register custom functions defined in go and calling these functions from your jade template.

**doctype**

the doctype shortcut does not support custom doctypes.

**Boolean Attribute**

the Boolean Attribute

    input(type='checkbox', checked=true && 'checked')

is not supported and no plans for support is currently in the pipeline, please log an issue if required.
use the ? conditional instead. Example:

    input(type='checkbox', checked=true ? 'checked')

**Unbuffered Code**
full javascript support for unbuffered code will not be supported as the template runs in go runtime.

Special cases are:

Defining a variable using var

	- var x = 5
    - var person = {name:"ben", age:5}

[Planned] Calling Unbuffered Functions defined in your code.

	- SomeGoFunction(5,64)
	  p This Content will be passed to the last argument of SomeGoFunction()
	p This content is outside of SomeGoFunctions scope.

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

## Using GoJade

**Installing gojade**


**Basic Usage**

Render a file

```golang
import "gojade"
.
.
.
data := map[string]interface{}{"name":"ben"}
gojade.RenderFile("index.jade",data)
```

# oxygen

![https://img.shields.io/github/v/tag/gromey/oxygen](https://img.shields.io/github/v/tag/gromey/oxygen)
![https://img.shields.io/github/license/gromey/oxygen](https://img.shields.io/github/license/gromey/oxygen)

`oxygen` is a library designed to create custom formatters like `JSON`, `XML`.

## Installation

`oxygen` can be installed like any other Go library through `go get`:

```console
go get github.com/gromey/oxygen
```

Or, if you are already using
[Go Modules](https://github.com/golang/go/wiki/Modules), you may specify a version number as well:

```console
go get github.com/gromey/oxygen@latest
```

## Getting Started

After you get the library, you must generate your type using the following command:

```console
go run github.com/gromey/oxygen/cmd/generate -n=name
```

In this command, you must specify a name of your new formatter. The name must contain only letters and be as simple as possible.

This command generates a package with the name specified in the generate command.
The package will contain two files `asserts.go` and `tag.go`.

**WARNING:** DO NOT EDIT `asserts.go`.

`tag.go` will contain the base implementation of your new formatter. You need to implement three functions **Parse**, **Encode** and **Decode**.

**Parse** function gets the value of the tag and a pointer to your tag structure,
here you need to parse the tag into a tag structure. If you don't use tags, just remove this method.

**Encode** function receives a value encoded into a byte array, if exists a tag struct and a field name,
here you can do additional encoding otherwise just remove this method.

**Decode** function receives an encoded data, if exists a tag struct and a field name, here you must find a byte array
representing a value for the current field and perform initial decoding if necessary before returning this byte array.  
You can change the input data and for the next field you will receive the data in a modified form,
however this will not affect the original data, since you are working with a copy of the data.

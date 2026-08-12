Typical usage
=============
```
thrift.exe --audit <oldFile> <newFile>
```
Example run
===========
```
> thrift.exe --audit vim-format-old.thrift vim-format-new.thrift
E:vim-format-new.thrift:4:15:Struct Field Requiredness Changed for Id = 3 in Delta
W:vim-format-new.thrift:5:15:Struct field name changed for Id = 4 in Delta
```

Diagnostics use Vim's `%t:%f:%l:%c:%m` error format.

Problems that the audit tool can catch
======================================
Errors
* Removing an enum value
* Changing the type of a struct field
* Changing the required-ness of a struct field
* Removing a struct field
* Adding a required struct field
* Adding a struct field 'in the middle'.  This usually indicates an old ID has been recycled
* Struct removed
* Oneway-ness change
* Return type change
* Missing function
* Missing service
* Change in service inheritance

Warnings
* Removing a language namespace declaration
* Changing a namespace
* Changing an enum value's name
* Removing an enum class
* Default value changed
* Struct field name change
* Removed constant
* Type of constant changed
* Value of constant changed

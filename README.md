miniweb
=======

A minimal website server using Markdown.

## Usage

1. Make your content

``` shell
$ cat <<EOS >index.md
hello world
===========

this is your first website on miniweb!

[another page](/page.html) is also available.
EOS

$ cat <<EOS >page.html
<!DOCTYPE html>

You can use <b>HTML</b> too.
```

2. Start miniweb.

``` shell
$ miniweb
```

Now you can see your website on <localhost:8000>.

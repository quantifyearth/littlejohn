# littlejohn

littlejohn is a small tool to let you run a simple command line tool in parallel. If you have a command of the form:

```
$ /bin/myprogram arg1 --arg2 foo --arg3 bar --arg4 wibble
```

And you want to run it many times, but with a different value for each of arg2 and arg3, you simply make a CSV file that looks like this:

```
--arg2, --arg3
foo, bar
flib, baz
fish, bop
```

The first line is the name of the arguments, and then each subsequent line is the values to pass for those arguments in successive runs of your program.

You then run littlejohn as follows:

```
$ littlejohn -j 4 -c myparams.csv /bin/myprogram -- --arg4 wibble
```

littlejohn will call `/bin/myprogram` multiple times, with up to 4 running concurrently in this instance (controller by -j). It will take arg2 and arg3 from the CSV file, and use a fixed arg4 for all invocations. Note the `--` in there: anything after the `--` is passed on to the child program as an argument directly.

So in this example it's the same as doing the following, with all three running at the same time:

```
$ /bin/myprogram arg1 --arg4 wibble --arg2 foo --arg3 bar
$ /bin/myprogram arg1 --arg4 wibble --arg2 flib --arg3 baz
$ /bin/myprogram arg1 --arg4 wibble --arg2 fish --arg3 bop
```

Note from this that fixed arguments are always provided to the target program first. This enables you to do something like this:

```
$ littlejohn -j 10 -c myparams.csv /bin/python3 -- ./myscript.py
```

Which would unroll to:

```
$ /bin/python3 ./myscript.py --arg2 foo --arg3 bar
$ /bin/python3 ./myscript.py --arg2 flib --arg3 baz
$ /bin/python3 ./myscript.py --arg2 fish --arg3 bop
```

If you need fixed arguments to go at the end, then the workaround is to add them to the CSV file in the order you want them to appear.

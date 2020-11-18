# PreSuf
[![Build Status](https://travis-ci.org/fpesce/presuf.svg?branch=master)](https://travis-ci.org/fpesce/presuf) 

A tool to plan for efficient exploration of suffix and prefix spaces when bruteforcing passwords.

When cracking a large number of passwords, if series of affixes patterns emerges, we could struggle to strategize how to efficiently crack them.
For example, let's say you only have time to explore all 8 characters long subspace (for 95 printable characters) of a single 4 letters prefix, and maybe 400 5-characters-long prefixes and 4000 6-characters-long prefixes. Your first approach would be to take the top prefixes of each... Except this would likely be inefficient because you will probably re-explore the same subspace several times.

If your currently cracked list shows a pattern where you have:
```
500 4 letters prefix miss
300 5 letters prefix miss.
250 4 letters prefix baby
```

It would be probably more advantageous to explore the 5 letters `miss.` prefix but then you should remove this pattern from the 4 letters prefixes, and this place `baby` as a better candidate.


## Running

This is ran on a sorted file (I usually prefer to `export LC_ALL=C` before all operation, incl. `sort`).
```
$ ./presub -input passwords.txt -min-prefix 4 -max-prefix 6 > last-results.txt
```

you could generate the same data for suffixes by reversing your passwords file first, a tool (reverse) is offered to do that.
```
$ ./reverse -input passwords.txt > reversed-passwords.txt
```

## Building

Run `make` or `make build` to compile your app.  This will use a Docker image
to build your app, with the current directory volume-mounted into place.  This
will store incremental state for the fastest possible build.  Run `make
all-build` to build for all architectures.

Run `make container` to build the container image.  It will calculate the image
tag based on the most recent git tag, and whether the repo is "dirty" since
that tag (see `make version`).  Run `make all-container` to build containers
for all supported architectures.

Run `make push` to push the container image to `REGISTRY`.  Run `make all-push`
to push the container images for all architectures.

Run `make clean` to clean up.

Run `make help` to get a list of available targets.

# Takehome Exercise - August 2018

## Setup

I used Go 1.10 on Mac OS X.

There's a simple Makefile with targets I used while working on this
exercise.

I used `dep` to manage a couple of dependencies from outside the
standard library. They're both for testing.

## General Approach

I bucketed values by minute. Hour sums are calculated by looking at
the most recent 60 buckets.

The structure is

    map[int64]map[string]int64

which means we look up by `minute`, and then by `key` to get a
`value`.

    minute -> key -> value

## Concurrency

I used the simplest correct approach: a global lock. Using the current
data structure, it may not be of much help to add more granular
locking.

That's because the likely access pattern is write-heavy, and all of
those writes will hit the most recent bucket! So the natural approach
of a lock per bucket is unlikely to help.

One alternative would be to flip the top-level datastructure: put
`key` on the outside, and then have 60 buckets per key:

    map[string]map[int64]int64

which would mean looking up by `key`, and then by `minute` to get a
`value`:

    key -> minute -> value

One disadvantage of putting the `key`s top-level is that
garbage-collecting expired buckets would be more work (since we'd have
to delete expired buckets under every key).

## Garbage Collecting Expired Values

As-is this program will grow over time. Old buckets will be held in
memory but never looked at.

One straightforward approach to dealing with this is to have the
`metrics` struct own a background goroutine that wakes up every minute
(or 5) and deletes expired buckets from the top-level map. (It would
want to keep track of how far back in time it checked the last time it
woke up.)

Note that we'd want to expose a `Closer` for clean shutdown if we're
managing a background goroutine.

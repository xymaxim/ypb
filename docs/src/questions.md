# Anticipated questions

Here are the answers to the most asked/anticipated questions.

## Why is the duration longer than the input interval?

Right now, downloaded media segments are merged as-is, without precise trimming
of the boundary segments to match the requested interval. The maximum positive
difference equals the duration of two segments, the start and end ones, and
depends on the type of live-streaming latency: up to 2 seconds (ultra-low
latency), 4 seconds (low), or 10 seconds (normal).

## Why is the duration shorter than the input interval?

Streams may contain gaps due to instability. A decrease in the duration of an
output excerpt (by seconds, minutes, or even hours) can result from: (a) the
start and/or end point falling within a gap, in which case the nearest available
segment is used instead, or (b) the excerpt containing one or more gaps
internally.
